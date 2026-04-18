package projects

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pi-memory/internal/db"
	"pi-memory/internal/migrations"
	"pi-memory/internal/util"
)

const defaultRootStrategy = "git-root-or-cwd"

var ErrAlreadyInitialized = errors.New("project already initialized")
var ErrNotFound = errors.New("project not found")

type InitInput struct {
	ProjectPath    string
	StorageBaseDir string
	ProjectName    string
}

type InitResult struct {
	ProjectID   string `json:"projectId"`
	ProjectDir  string `json:"projectDir"`
	ProjectFile string `json:"projectFile"`
	DBPath      string `json:"dbPath"`
	Created     bool   `json:"created"`
}

type GetResult struct {
	Initialized bool             `json:"initialized"`
	Project     *ProjectMetadata `json:"project,omitempty"`
}

type StatusResult struct {
	Initialized         bool   `json:"initialized"`
	ProjectID           string `json:"projectId,omitempty"`
	DBPath              string `json:"dbPath,omitempty"`
	ActiveMemoryCount   int    `json:"activeMemoryCount"`
	TrackedSessionCount int    `json:"trackedSessionCount"`
	LastIngestedAt      string `json:"lastIngestedAt"`
}

func Init(input InitInput) (*InitResult, error) {
	projectPath, storageBaseDir, err := resolvePaths(input.ProjectPath, input.StorageBaseDir)
	if err != nil {
		return nil, err
	}
	registry, registryPath, err := loadRegistry(storageBaseDir)
	if err != nil {
		return nil, err
	}
	if existing := findByProjectPath(registry, projectPath); existing != nil {
		return &InitResult{ProjectID: existing.ProjectID, ProjectDir: existing.ProjectDir, ProjectFile: existing.ProjectFile, DBPath: existing.DBPath, Created: false}, ErrAlreadyInitialized
	}

	name := strings.TrimSpace(input.ProjectName)
	if name == "" {
		name = filepath.Base(projectPath)
	}
	slug := util.Slugify(name)
	hash := util.ShortHash(projectPath)
	projectID := fmt.Sprintf("%s-%s", slug, hash)
	projectDir := filepath.Join(storageBaseDir, projectID)
	projectFile := filepath.Join(projectDir, "project.json")
	dbPath := filepath.Join(projectDir, "memory.db")
	now := now()

	metadata := ProjectMetadata{
		Version:             1,
		ProjectID:           projectID,
		Name:                name,
		Slug:                slug,
		Hash:                hash,
		ProjectPath:         projectPath,
		ProjectRootStrategy: defaultRootStrategy,
		ProjectDir:          projectDir,
		DBPath:              dbPath,
		CreatedAt:           now,
		UpdatedAt:           now,
		Status:              "active",
	}

	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return nil, err
	}
	if err := writeJSON(projectFile, metadata); err != nil {
		return nil, err
	}

	sqldb, err := db.Open(dbPath)
	if err != nil {
		return nil, err
	}
	defer sqldb.Close()
	if err := migrations.Apply(sqldb); err != nil {
		return nil, err
	}

	entry := RegistryEntry{
		ProjectID:   projectID,
		Name:        name,
		Slug:        slug,
		Hash:        hash,
		ProjectPath: projectPath,
		ProjectDir:  projectDir,
		ProjectFile: projectFile,
		DBPath:      dbPath,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	registry.BaseStorageDir = storageBaseDir
	registry.Projects = append(registry.Projects, entry)
	if err := writeJSON(registryPath, registry); err != nil {
		return nil, err
	}

	return &InitResult{ProjectID: projectID, ProjectDir: projectDir, ProjectFile: projectFile, DBPath: dbPath, Created: true}, nil
}

func Get(projectPath, storageBaseDir string) (*GetResult, error) {
	projectPath, storageBaseDir, err := resolvePaths(projectPath, storageBaseDir)
	if err != nil {
		return nil, err
	}
	registry, _, err := loadRegistry(storageBaseDir)
	if err != nil {
		return nil, err
	}
	entry := findByProjectPath(registry, projectPath)
	if entry == nil {
		return &GetResult{Initialized: false}, nil
	}
	metadata, err := readProjectFile(entry.ProjectFile)
	if err != nil {
		return nil, err
	}
	return &GetResult{Initialized: true, Project: metadata}, nil
}

func Status(projectPath, storageBaseDir string) (*StatusResult, error) {
	res, err := Get(projectPath, storageBaseDir)
	if err != nil {
		return nil, err
	}
	if !res.Initialized || res.Project == nil {
		return &StatusResult{Initialized: false}, nil
	}

	sqldb, err := db.Open(res.Project.DBPath)
	if err != nil {
		return nil, err
	}
	defer sqldb.Close()

	activeCount, err := scalarInt(sqldb, `SELECT COUNT(*) FROM memory_items WHERE status = 'active'`)
	if err != nil {
		return nil, err
	}
	trackedCount, err := scalarInt(sqldb, `SELECT COUNT(*) FROM tracked_sessions WHERE status = 'active'`)
	if err != nil {
		return nil, err
	}
	lastIngestedAt, err := scalarString(sqldb, `SELECT COALESCE(MAX(last_ingested_at), '') FROM ingestion_state`)
	if err != nil {
		return nil, err
	}

	return &StatusResult{
		Initialized:         true,
		ProjectID:           res.Project.ProjectID,
		DBPath:              res.Project.DBPath,
		ActiveMemoryCount:   activeCount,
		TrackedSessionCount: trackedCount,
		LastIngestedAt:      lastIngestedAt,
	}, nil
}

func resolvePaths(projectPath, storageBaseDir string) (string, string, error) {
	canonicalProjectPath, err := util.CanonicalPath(projectPath)
	if err != nil {
		return "", "", err
	}
	if info, err := os.Stat(canonicalProjectPath); err != nil || !info.IsDir() {
		if err == nil {
			err = fmt.Errorf("project path is not a directory")
		}
		return "", "", err
	}
	baseDir := storageBaseDir
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "~/.pi-memory"
	}
	canonicalBaseDir, err := util.CanonicalPath(baseDir)
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(canonicalBaseDir, 0o755); err != nil {
		return "", "", err
	}
	return canonicalProjectPath, canonicalBaseDir, nil
}

func loadRegistry(storageBaseDir string) (*Registry, string, error) {
	registryPath := filepath.Join(storageBaseDir, "projects.json")
	if _, err := os.Stat(registryPath); errors.Is(err, os.ErrNotExist) {
		return &Registry{Version: 1, BaseStorageDir: storageBaseDir, Projects: []RegistryEntry{}}, registryPath, nil
	} else if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, "", err
	}
	var registry Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, "", err
	}
	if registry.Version == 0 {
		registry.Version = 1
	}
	if registry.Projects == nil {
		registry.Projects = []RegistryEntry{}
	}
	if registry.BaseStorageDir == "" {
		registry.BaseStorageDir = storageBaseDir
	}
	return &registry, registryPath, nil
}

func findByProjectPath(registry *Registry, projectPath string) *RegistryEntry {
	for i := range registry.Projects {
		if registry.Projects[i].ProjectPath == projectPath {
			return &registry.Projects[i]
		}
	}
	return nil
}

func readProjectFile(path string) (*ProjectMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var metadata ProjectMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func scalarInt(db *sql.DB, query string) (int, error) {
	var value int
	err := db.QueryRow(query).Scan(&value)
	return value, err
}

func scalarString(db *sql.DB, query string) (string, error) {
	var value string
	err := db.QueryRow(query).Scan(&value)
	return value, err
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}
