package sessions

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"pi-memory/internal/ingest"
	"pi-memory/internal/memories"
	"pi-memory/internal/projects"
	"pi-memory/internal/util"
)

type DiscoverInput struct {
	Project        *projects.ProjectMetadata
	SessionDir     string
	ActiveFileOnly string
}

type TrackedSession struct {
	SessionFile string
	FirstSeenAt string
	LastSeenAt  string
	Status      string
}

type IngestResult struct {
	RunID                     string `json:"runId"`
	TrackedSessionsDiscovered int    `json:"trackedSessionsDiscovered"`
	SessionFilesProcessed     int    `json:"sessionFilesProcessed"`
	EntriesSeen               int    `json:"entriesSeen"`
	CandidatesFound           int    `json:"candidatesFound"`
	MemoriesCreated           int    `json:"memoriesCreated"`
	MemoriesUpdated           int    `json:"memoriesUpdated"`
	MemoriesIgnored           int    `json:"memoriesIgnored"`
	LastIngestedAt            string `json:"lastIngestedAt"`
}

type IngestInput struct {
	Project           *projects.ProjectMetadata
	SessionDir        string
	Trigger           string
	ActiveSessionFile string
}

func Ingest(db *sql.DB, input IngestInput) (*IngestResult, error) {
	discovered, err := DiscoverAndTrack(db, DiscoverInput{
		Project:        input.Project,
		SessionDir:     input.SessionDir,
		ActiveFileOnly: input.ActiveSessionFile,
	})
	if err != nil {
		return nil, err
	}

	runID := "run_" + uuid.NewString()
	startedAt := now()
	if _, err := db.Exec(`INSERT INTO ingestion_runs (run_id, started_at, status, trigger) VALUES (?, ?, 'running', ?)`, runID, startedAt, input.Trigger); err != nil {
		return nil, err
	}

	result := &IngestResult{
		RunID:                     runID,
		TrackedSessionsDiscovered: len(discovered),
		LastIngestedAt:            startedAt,
	}

	for _, tracked := range discovered {
		processed, entriesSeen, candidatesFound, memoriesCreated, memoriesUpdated, memoriesIgnored, lastEntryID, lastEntryTimestamp, err := processSessionFile(db, input.Project.ProjectID, runID, tracked.SessionFile)
		if err != nil {
			_, _ = db.Exec(`UPDATE ingestion_runs SET status = 'failed', finished_at = ?, error_message = ? WHERE run_id = ?`, now(), err.Error(), runID)
			return nil, err
		}
		if processed {
			result.SessionFilesProcessed++
		}
		result.EntriesSeen += entriesSeen
		result.CandidatesFound += candidatesFound
		result.MemoriesCreated += memoriesCreated
		result.MemoriesUpdated += memoriesUpdated
		result.MemoriesIgnored += memoriesIgnored
		if processed {
			timestamp := now()
			if _, err := db.Exec(`
				INSERT INTO ingestion_state (session_file, last_entry_id, last_entry_timestamp, last_ingested_at, last_run_id)
				VALUES (?, ?, ?, ?, ?)
				ON CONFLICT(session_file) DO UPDATE SET
				  last_entry_id = excluded.last_entry_id,
				  last_entry_timestamp = excluded.last_entry_timestamp,
				  last_ingested_at = excluded.last_ingested_at,
				  last_run_id = excluded.last_run_id
			`, tracked.SessionFile, lastEntryID, lastEntryTimestamp, timestamp, runID); err != nil {
				return nil, err
			}
			if _, err := db.Exec(`UPDATE tracked_sessions SET last_ingested_at = ?, last_seen_at = ?, status = 'active' WHERE session_file = ?`, timestamp, timestamp, tracked.SessionFile); err != nil {
				return nil, err
			}
			result.LastIngestedAt = timestamp
		}
	}

	if _, err := db.Exec(`
		UPDATE ingestion_runs
		SET finished_at = ?, status = 'completed', entries_seen = ?, candidates_found = ?, memories_created = ?, memories_updated = ?, memories_ignored = ?
		WHERE run_id = ?
	`, now(), result.EntriesSeen, result.CandidatesFound, result.MemoriesCreated, result.MemoriesUpdated, result.MemoriesIgnored, runID); err != nil {
		return nil, err
	}

	return result, nil
}

func DiscoverAndTrack(db *sql.DB, input DiscoverInput) ([]TrackedSession, error) {
	sessionDir, err := resolveSessionProjectDir(input.Project.ProjectPath, input.SessionDir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(sessionDir)
	if errors.Is(err, os.ErrNotExist) {
		return []TrackedSession{}, nil
	}
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		files = append(files, filepath.Join(sessionDir, entry.Name()))
	}
	sort.Strings(files)

	if input.ActiveFileOnly != "" {
		canonical, err := util.CanonicalPath(input.ActiveFileOnly)
		if err != nil {
			return nil, err
		}
		files = filterToActive(files, canonical)
	}

	nowValue := now()
	tracked := make([]TrackedSession, 0, len(files))
	for _, file := range files {
		canonical, err := util.CanonicalPath(file)
		if err != nil {
			return nil, err
		}
		if _, err := db.Exec(`
			INSERT INTO tracked_sessions (session_file, project_id, first_seen_at, last_seen_at, status)
			VALUES (?, ?, ?, ?, 'active')
			ON CONFLICT(session_file) DO UPDATE SET
			  last_seen_at = excluded.last_seen_at,
			  status = 'active'
		`, canonical, input.Project.ProjectID, nowValue, nowValue); err != nil {
			return nil, err
		}
		tracked = append(tracked, TrackedSession{SessionFile: canonical, FirstSeenAt: nowValue, LastSeenAt: nowValue, Status: "active"})
	}

	return tracked, nil
}

func resolveSessionProjectDir(projectPath, sessionDirOverride string) (string, error) {
	baseDir := sessionDirOverride
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "~/.pi/agent/sessions"
	}
	canonicalBaseDir, err := util.CanonicalPath(baseDir)
	if err != nil {
		return "", err
	}
	normalizedProjectPath := strings.TrimPrefix(projectPath, string(filepath.Separator))
	sessionFolder := "--" + strings.ReplaceAll(normalizedProjectPath, string(filepath.Separator), "-") + "--"
	return filepath.Join(canonicalBaseDir, sessionFolder), nil
}

func filterToActive(files []string, active string) []string {
	for _, file := range files {
		if file == active {
			return []string{file}
		}
	}
	return files
}

func processSessionFile(db *sql.DB, projectID, runID, sessionFile string) (processed bool, entriesSeen int, candidatesFound int, memoriesCreated int, memoriesUpdated int, memoriesIgnored int, lastEntryID string, lastEntryTimestamp string, err error) {
	lastCheckpointID, err := existingCheckpoint(db, sessionFile)
	if err != nil {
		return false, 0, 0, 0, 0, 0, "", "", err
	}

	file, err := os.Open(sessionFile)
	if err != nil {
		return false, 0, 0, 0, 0, 0, "", "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	checkpointFound := lastCheckpointID == ""

	for scanner.Scan() {
		line := scanner.Bytes()
		var entry ingest.SessionEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.ID == "" {
			continue
		}
		if !checkpointFound {
			if entry.ID == lastCheckpointID {
				checkpointFound = true
			}
			continue
		}
		entriesSeen++
		processed = true
		lastEntryID = entry.ID
		lastEntryTimestamp = entry.Timestamp
		candidates := ingest.ExtractCandidates(sessionFile, entry)
		candidatesFound += len(candidates)
		for _, candidate := range candidates {
			outcome, upsertErr := memories.UpsertCandidate(db, projectID, candidate)
			if upsertErr != nil {
				return false, 0, 0, 0, 0, 0, "", "", upsertErr
			}
			if outcome.Created {
				memoriesCreated++
			}
			if outcome.Updated {
				memoriesUpdated++
			}
			if outcome.Ignored {
				memoriesIgnored++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return false, 0, 0, 0, 0, 0, "", "", err
	}

	if lastCheckpointID != "" && !checkpointFound {
		return reprocessWholeSession(db, projectID, sessionFile)
	}

	if _, err := db.Exec(`UPDATE ingestion_runs SET session_file = COALESCE(session_file, ?) WHERE run_id = ?`, sessionFile, runID); err != nil {
		return false, 0, 0, 0, 0, 0, "", "", err
	}

	return processed, entriesSeen, candidatesFound, memoriesCreated, memoriesUpdated, memoriesIgnored, lastEntryID, lastEntryTimestamp, nil
}

func reprocessWholeSession(db *sql.DB, projectID, sessionFile string) (bool, int, int, int, int, int, string, string, error) {
	file, err := os.Open(sessionFile)
	if err != nil {
		return false, 0, 0, 0, 0, 0, "", "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	entriesSeen := 0
	candidatesFound := 0
	memoriesCreated := 0
	memoriesUpdated := 0
	memoriesIgnored := 0
	lastEntryID := ""
	lastEntryTimestamp := ""
	for scanner.Scan() {
		var entry ingest.SessionEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if entry.ID == "" {
			continue
		}
		entriesSeen++
		lastEntryID = entry.ID
		lastEntryTimestamp = entry.Timestamp
		candidates := ingest.ExtractCandidates(sessionFile, entry)
		candidatesFound += len(candidates)
		for _, candidate := range candidates {
			outcome, upsertErr := memories.UpsertCandidate(db, projectID, candidate)
			if upsertErr != nil {
				return false, 0, 0, 0, 0, 0, "", "", upsertErr
			}
			if outcome.Created {
				memoriesCreated++
			}
			if outcome.Updated {
				memoriesUpdated++
			}
			if outcome.Ignored {
				memoriesIgnored++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return false, 0, 0, 0, 0, 0, "", "", err
	}
	return entriesSeen > 0, entriesSeen, candidatesFound, memoriesCreated, memoriesUpdated, memoriesIgnored, lastEntryID, lastEntryTimestamp, nil
}

func existingCheckpoint(db *sql.DB, sessionFile string) (string, error) {
	var lastEntryID sql.NullString
	err := db.QueryRow(`SELECT last_entry_id FROM ingestion_state WHERE session_file = ?`, sessionFile).Scan(&lastEntryID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !lastEntryID.Valid {
		return "", nil
	}
	return lastEntryID.String, nil
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func DebugProjectSessionDir(projectPath, sessionDirOverride string) (string, error) {
	return resolveSessionProjectDir(projectPath, sessionDirOverride)
}

func ExplainProjectSessionDir(projectPath string) string {
	normalizedProjectPath := strings.TrimPrefix(projectPath, string(filepath.Separator))
	return fmt.Sprintf("--%s--", strings.ReplaceAll(normalizedProjectPath, string(filepath.Separator), "-"))
}
