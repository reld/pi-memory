package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"pi-memory/internal/api"
	"pi-memory/internal/projects"
)

type projectPayload struct {
	ProjectPath    string `json:"projectPath"`
	StorageBaseDir string `json:"storageBaseDir"`
	ProjectName    string `json:"projectName,omitempty"`
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

func responsePtr(resp api.Response) *api.Response {
	return &resp
}

func writeProcessError(code string, err error) {
	_, _ = fmt.Fprintf(os.Stderr, "%s: %v\n", code, err)
}
