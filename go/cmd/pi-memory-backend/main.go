package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"pi-memory/internal/api"
	"pi-memory/internal/db"
	"pi-memory/internal/memories"
	"pi-memory/internal/projects"
	"pi-memory/internal/sessions"
)

type projectPayload struct {
	ProjectPath    string `json:"projectPath"`
	StorageBaseDir string `json:"storageBaseDir"`
	ProjectName    string `json:"projectName,omitempty"`
}

type ingestPayload struct {
	ProjectPath       string `json:"projectPath"`
	StorageBaseDir    string `json:"storageBaseDir"`
	SessionDir        string `json:"sessionDir,omitempty"`
	Trigger           string `json:"trigger,omitempty"`
	ActiveSessionFile string `json:"activeSessionFile,omitempty"`
}

type listMemoriesPayload struct {
	ProjectPath    string `json:"projectPath"`
	StorageBaseDir string `json:"storageBaseDir"`
	Status         string `json:"status,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

type searchMemoriesPayload struct {
	ProjectPath    string `json:"projectPath"`
	StorageBaseDir string `json:"storageBaseDir"`
	Query          string `json:"query"`
	Limit          int    `json:"limit,omitempty"`
}

type searchSessionsPayload struct {
	ProjectPath    string `json:"projectPath"`
	StorageBaseDir string `json:"storageBaseDir"`
	SessionDir     string `json:"sessionDir,omitempty"`
	Query          string `json:"query"`
	Limit          int    `json:"limit,omitempty"`
}

type recallMemoriesPayload struct {
	ProjectPath    string `json:"projectPath"`
	StorageBaseDir string `json:"storageBaseDir"`
	Limit          int    `json:"limit,omitempty"`
}

type forgetMemoryPayload struct {
	ProjectPath    string `json:"projectPath"`
	StorageBaseDir string `json:"storageBaseDir"`
	MemoryID       string `json:"memoryId"`
	Mode           string `json:"mode,omitempty"`
}

type rememberMemoryPayload struct {
	ProjectPath    string `json:"projectPath"`
	StorageBaseDir string `json:"storageBaseDir"`
	Text           string `json:"text"`
}

func main() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		writeProcessError("READ_STDIN_FAILED", err)
		os.Exit(1)
	}

	var req api.Request
	if err := json.Unmarshal(input, &req); err != nil {
		debugLogf("invalid request json: %v", err)
		_ = json.NewEncoder(os.Stdout).Encode(api.Failure("INVALID_REQUEST", "Failed to parse request JSON", nil))
		return
	}

	debugLogf("command=%s version=%d", req.Command, req.Version)
	resp := dispatch(req)
	if resp.OK {
		debugLogf("command=%s ok", req.Command)
	} else if resp.Error != nil {
		debugLogf("command=%s error code=%s message=%s", req.Command, resp.Error.Code, resp.Error.Message)
	}
	if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		writeProcessError("WRITE_STDOUT_FAILED", err)
		os.Exit(1)
	}
}

func dispatch(req api.Request) api.Response {
	switch req.Command {
	case "health":
		return api.Success(map[string]any{
			"message": "pi-memory backend scaffold is running",
			"version": req.Version,
		})
	case "init_project":
		payload, fail := decodeProjectPayload(req.Payload)
		if fail != nil {
			return *fail
		}
		result, err := projects.Init(projects.InitInput{
			ProjectPath:    payload.ProjectPath,
			StorageBaseDir: payload.StorageBaseDir,
			ProjectName:    payload.ProjectName,
		})
		if err != nil {
			if err == projects.ErrAlreadyInitialized {
				return api.Failure("PROJECT_ALREADY_INITIALIZED", err.Error(), map[string]any{"projectId": result.ProjectID})
			}
			return api.Failure("INIT_FAILED", err.Error(), nil)
		}
		return api.Success(result)
	case "get_project":
		payload, fail := decodeProjectPayload(req.Payload)
		if fail != nil {
			return *fail
		}
		result, err := projects.Get(payload.ProjectPath, payload.StorageBaseDir)
		if err != nil {
			return api.Failure("PROJECT_LOOKUP_FAILED", err.Error(), nil)
		}
		return api.Success(result)
	case "project_status":
		payload, fail := decodeProjectPayload(req.Payload)
		if fail != nil {
			return *fail
		}
		result, err := projects.Status(payload.ProjectPath, payload.StorageBaseDir)
		if err != nil {
			return api.Failure("PROJECT_STATUS_FAILED", err.Error(), nil)
		}
		return api.Success(result)
	case "ingest_sessions":
		payload, fail := decodeIngestPayload(req.Payload)
		if fail != nil {
			return *fail
		}
		projectResult, err := projects.Get(payload.ProjectPath, payload.StorageBaseDir)
		if err != nil {
			return api.Failure("PROJECT_LOOKUP_FAILED", err.Error(), nil)
		}
		if !projectResult.Initialized || projectResult.Project == nil {
			return api.Failure("PROJECT_NOT_INITIALIZED", "Project is not initialized", nil)
		}
		sqldb, err := db.Open(projectResult.Project.DBPath)
		if err != nil {
			return api.Failure("DB_ERROR", err.Error(), nil)
		}
		defer sqldb.Close()
		result, err := sessions.Ingest(sqldb, sessions.IngestInput{
			Project:           projectResult.Project,
			SessionDir:        payload.SessionDir,
			Trigger:           defaultTrigger(payload.Trigger),
			ActiveSessionFile: payload.ActiveSessionFile,
		})
		if err != nil {
			return api.Failure("INGEST_FAILED", err.Error(), nil)
		}
		return api.Success(result)
	case "rebuild_project_memory":
		payload, fail := decodeIngestPayload(req.Payload)
		if fail != nil {
			return *fail
		}
		projectResult, err := projects.Get(payload.ProjectPath, payload.StorageBaseDir)
		if err != nil {
			return api.Failure("PROJECT_LOOKUP_FAILED", err.Error(), nil)
		}
		if !projectResult.Initialized || projectResult.Project == nil {
			return api.Failure("PROJECT_NOT_INITIALIZED", "Project is not initialized", nil)
		}
		sqldb, err := db.Open(projectResult.Project.DBPath)
		if err != nil {
			return api.Failure("DB_ERROR", err.Error(), nil)
		}
		defer sqldb.Close()
		result, err := sessions.Rebuild(sqldb, sessions.IngestInput{
			Project:           projectResult.Project,
			SessionDir:        payload.SessionDir,
			Trigger:           defaultTrigger(payload.Trigger),
			ActiveSessionFile: payload.ActiveSessionFile,
		})
		if err != nil {
			return api.Failure("INGEST_FAILED", err.Error(), nil)
		}
		return api.Success(result)
	case "list_memories":
		payload, fail := decodeListMemoriesPayload(req.Payload)
		if fail != nil {
			return *fail
		}
		projectResult, response := openProjectDB(payload.ProjectPath, payload.StorageBaseDir)
		if response != nil {
			return *response
		}
		defer projectResult.DB.Close()
		items, err := memories.List(projectResult.DB, projectResult.Project.ProjectID, payload.Status, payload.Limit)
		if err != nil {
			return api.Failure("SEARCH_FAILED", err.Error(), nil)
		}
		return api.Success(map[string]any{"items": items})
	case "search_memories":
		payload, fail := decodeSearchMemoriesPayload(req.Payload)
		if fail != nil {
			return *fail
		}
		projectResult, response := openProjectDB(payload.ProjectPath, payload.StorageBaseDir)
		if response != nil {
			return *response
		}
		defer projectResult.DB.Close()
		items, err := memories.Search(projectResult.DB, projectResult.Project.ProjectID, payload.Query, payload.Limit)
		if err != nil {
			return api.Failure("SEARCH_FAILED", err.Error(), nil)
		}
		return api.Success(map[string]any{"items": items})
	case "recall_memories":
		payload, fail := decodeRecallMemoriesPayload(req.Payload)
		if fail != nil {
			return *fail
		}
		projectResult, response := openProjectDB(payload.ProjectPath, payload.StorageBaseDir)
		if response != nil {
			return *response
		}
		defer projectResult.DB.Close()
		items, err := memories.Recall(projectResult.DB, projectResult.Project.ProjectID, payload.Limit)
		if err != nil {
			return api.Failure("SEARCH_FAILED", err.Error(), nil)
		}
		return api.Success(map[string]any{"items": items})
	case "search_sessions":
		payload, fail := decodeSearchSessionsPayload(req.Payload)
		if fail != nil {
			return *fail
		}
		projectResult, response := openProjectDB(payload.ProjectPath, payload.StorageBaseDir)
		if response != nil {
			return *response
		}
		defer projectResult.DB.Close()
		items, err := sessions.Search(projectResult.DB, sessions.SearchSessionsInput{
			Project:    projectResult.Project,
			SessionDir: payload.SessionDir,
			Query:      payload.Query,
			Limit:      payload.Limit,
		})
		if err != nil {
			return api.Failure("SEARCH_FAILED", err.Error(), nil)
		}
		return api.Success(map[string]any{"items": items})
	case "forget_memory":
		payload, fail := decodeForgetMemoryPayload(req.Payload)
		if fail != nil {
			return *fail
		}
		projectResult, response := openProjectDB(payload.ProjectPath, payload.StorageBaseDir)
		if response != nil {
			return *response
		}
		defer projectResult.DB.Close()
		result, err := memories.Forget(projectResult.DB, projectResult.Project.ProjectID, payload.MemoryID, payload.Mode)
		if err != nil {
			if err == memories.ErrMemoryNotFound {
				return api.Failure("MEMORY_NOT_FOUND", err.Error(), nil)
			}
			return api.Failure("DB_ERROR", err.Error(), nil)
		}
		return api.Success(result)
	case "remember_memory":
		payload, fail := decodeRememberMemoryPayload(req.Payload)
		if fail != nil {
			return *fail
		}
		projectResult, response := openProjectDB(payload.ProjectPath, payload.StorageBaseDir)
		if response != nil {
			return *response
		}
		defer projectResult.DB.Close()
		result, err := memories.Remember(projectResult.DB, projectResult.Project.ProjectID, payload.Text)
		if err != nil {
			return api.Failure("DB_ERROR", err.Error(), nil)
		}
		return api.Success(result)
	default:
		return api.Failure("COMMAND_NOT_IMPLEMENTED", fmt.Sprintf("Command %q is not implemented yet", req.Command), nil)
	}
}

func decodeProjectPayload(raw json.RawMessage) (*projectPayload, *api.Response) {
	var payload projectPayload
	if len(raw) == 0 {
		return nil, responsePtr(api.Failure("INVALID_PAYLOAD", "Missing payload", nil))
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, responsePtr(api.Failure("INVALID_PAYLOAD", "Failed to parse payload", nil))
	}
	if payload.ProjectPath == "" {
		return nil, responsePtr(api.Failure("INVALID_PROJECT_PATH", "projectPath is required", nil))
	}
	return &payload, nil
}

func decodeIngestPayload(raw json.RawMessage) (*ingestPayload, *api.Response) {
	var payload ingestPayload
	if len(raw) == 0 {
		return nil, responsePtr(api.Failure("INVALID_PAYLOAD", "Missing payload", nil))
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, responsePtr(api.Failure("INVALID_PAYLOAD", "Failed to parse payload", nil))
	}
	if payload.ProjectPath == "" {
		return nil, responsePtr(api.Failure("INVALID_PROJECT_PATH", "projectPath is required", nil))
	}
	return &payload, nil
}

type openedProject struct {
	Project *projects.ProjectMetadata
	DB      *sql.DB
}

func decodeListMemoriesPayload(raw json.RawMessage) (*listMemoriesPayload, *api.Response) {
	var payload listMemoriesPayload
	if len(raw) == 0 {
		return nil, responsePtr(api.Failure("INVALID_PAYLOAD", "Missing payload", nil))
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, responsePtr(api.Failure("INVALID_PAYLOAD", "Failed to parse payload", nil))
	}
	if payload.ProjectPath == "" {
		return nil, responsePtr(api.Failure("INVALID_PROJECT_PATH", "projectPath is required", nil))
	}
	return &payload, nil
}

func decodeSearchMemoriesPayload(raw json.RawMessage) (*searchMemoriesPayload, *api.Response) {
	var payload searchMemoriesPayload
	if len(raw) == 0 {
		return nil, responsePtr(api.Failure("INVALID_PAYLOAD", "Missing payload", nil))
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, responsePtr(api.Failure("INVALID_PAYLOAD", "Failed to parse payload", nil))
	}
	if payload.ProjectPath == "" {
		return nil, responsePtr(api.Failure("INVALID_PROJECT_PATH", "projectPath is required", nil))
	}
	if payload.Query == "" {
		return nil, responsePtr(api.Failure("INVALID_QUERY", "query is required", nil))
	}
	return &payload, nil
}

func decodeSearchSessionsPayload(raw json.RawMessage) (*searchSessionsPayload, *api.Response) {
	var payload searchSessionsPayload
	if len(raw) == 0 {
		return nil, responsePtr(api.Failure("INVALID_PAYLOAD", "Missing payload", nil))
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, responsePtr(api.Failure("INVALID_PAYLOAD", "Failed to parse payload", nil))
	}
	if payload.ProjectPath == "" {
		return nil, responsePtr(api.Failure("INVALID_PROJECT_PATH", "projectPath is required", nil))
	}
	if payload.Query == "" {
		return nil, responsePtr(api.Failure("INVALID_QUERY", "query is required", nil))
	}
	return &payload, nil
}

func decodeRememberMemoryPayload(raw json.RawMessage) (*rememberMemoryPayload, *api.Response) {
	var payload rememberMemoryPayload
	if len(raw) == 0 {
		return nil, responsePtr(api.Failure("INVALID_PAYLOAD", "Missing payload", nil))
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, responsePtr(api.Failure("INVALID_PAYLOAD", "Failed to parse payload", nil))
	}
	if payload.ProjectPath == "" {
		return nil, responsePtr(api.Failure("INVALID_PROJECT_PATH", "projectPath is required", nil))
	}
	if payload.Text == "" {
		return nil, responsePtr(api.Failure("INVALID_TEXT", "text is required", nil))
	}
	return &payload, nil
}

func decodeForgetMemoryPayload(raw json.RawMessage) (*forgetMemoryPayload, *api.Response) {
	var payload forgetMemoryPayload
	if len(raw) == 0 {
		return nil, responsePtr(api.Failure("INVALID_PAYLOAD", "Missing payload", nil))
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, responsePtr(api.Failure("INVALID_PAYLOAD", "Failed to parse payload", nil))
	}
	if payload.ProjectPath == "" {
		return nil, responsePtr(api.Failure("INVALID_PROJECT_PATH", "projectPath is required", nil))
	}
	if payload.MemoryID == "" {
		return nil, responsePtr(api.Failure("INVALID_MEMORY_ID", "memoryId is required", nil))
	}
	return &payload, nil
}

func decodeRecallMemoriesPayload(raw json.RawMessage) (*recallMemoriesPayload, *api.Response) {
	var payload recallMemoriesPayload
	if len(raw) == 0 {
		return nil, responsePtr(api.Failure("INVALID_PAYLOAD", "Missing payload", nil))
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, responsePtr(api.Failure("INVALID_PAYLOAD", "Failed to parse payload", nil))
	}
	if payload.ProjectPath == "" {
		return nil, responsePtr(api.Failure("INVALID_PROJECT_PATH", "projectPath is required", nil))
	}
	return &payload, nil
}

func openProjectDB(projectPath, storageBaseDir string) (*openedProject, *api.Response) {
	projectResult, err := projects.Get(projectPath, storageBaseDir)
	if err != nil {
		return nil, responsePtr(api.Failure("PROJECT_LOOKUP_FAILED", err.Error(), nil))
	}
	if !projectResult.Initialized || projectResult.Project == nil {
		return nil, responsePtr(api.Failure("PROJECT_NOT_INITIALIZED", "Project is not initialized", nil))
	}
	sqldb, err := db.Open(projectResult.Project.DBPath)
	if err != nil {
		return nil, responsePtr(api.Failure("DB_ERROR", err.Error(), nil))
	}
	return &openedProject{Project: projectResult.Project, DB: sqldb}, nil
}

func defaultTrigger(trigger string) string {
	if trigger == "" {
		return "manual"
	}
	return trigger
}

func responsePtr(resp api.Response) *api.Response {
	return &resp
}

func writeProcessError(code string, err error) {
	_, _ = fmt.Fprintf(os.Stderr, "%s: %v\n", code, err)
}

func debugLogf(format string, args ...any) {
	if !debugEnabled() {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "[%s] pi-memory: %s\n", time.Now().UTC().Format(time.RFC3339), fmt.Sprintf(format, args...))
}

func debugEnabled() bool {
	value := os.Getenv("PI_MEMORY_DEBUG")
	return value == "1" || value == "true" || value == "TRUE" || value == "yes" || value == "on"
}
