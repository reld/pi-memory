package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"pi-memory/internal/api"
	"pi-memory/internal/db"
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

func main() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		writeProcessError("READ_STDIN_FAILED", err)
		os.Exit(1)
	}

	var req api.Request
	if err := json.Unmarshal(input, &req); err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(api.Failure("INVALID_REQUEST", "Failed to parse request JSON", nil))
		return
	}

	resp := dispatch(req)
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
