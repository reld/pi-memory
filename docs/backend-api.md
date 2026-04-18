# Backend API

This document describes the Go backend API used by the TypeScript extension.

## Transport

The backend uses **JSON-over-stdio**.

The TypeScript extension:
- spawns the backend binary
- writes one JSON request to stdin
- reads one JSON response from stdout
- reads debug/error logs from stderr

Primary implementation files:
- `go/cmd/pi-memory-backend/main.go`
- `src/extension/services/backend.ts`

## Request envelope

All requests use this shape:

```json
{
  "version": 1,
  "command": "project_status",
  "payload": {
    "projectPath": "/absolute/project/path",
    "storageBaseDir": "/Users/me/.pi-memory"
  }
}
```

Fields:
- `version: 1`
- `command: string`
- `payload: object`

## Response envelope

### Success

```json
{
  "ok": true,
  "result": {}
}
```

### Failure

```json
{
  "ok": false,
  "error": {
    "code": "PROJECT_NOT_INITIALIZED",
    "message": "Project is not initialized",
    "details": {}
  }
}
```

Fields:
- `ok: boolean`
- `result: any` on success
- `error.code: string` on failure
- `error.message: string` on failure
- `error.details?: object` on failure

## Commands

### `health`

Minimal health probe.

Request payload:
- none required

Success result:

```json
{
  "message": "pi-memory backend scaffold is running",
  "version": 1
}
```

---

### `init_project`

Initializes project metadata and the project database.

Payload:
- `projectPath: string` required
- `storageBaseDir: string` optional in decoder, but expected by implementation
- `projectName?: string`

Example:

```json
{
  "version": 1,
  "command": "init_project",
  "payload": {
    "projectPath": "/Users/me/Code/pi-memory",
    "storageBaseDir": "/Users/me/.pi-memory"
  }
}
```

Success result:
- `projectId: string`
- `projectDir: string`
- `projectFile: string`
- `dbPath: string`
- `created: boolean`

Possible errors:
- `INVALID_PAYLOAD`
- `INVALID_PROJECT_PATH`
- `PROJECT_ALREADY_INITIALIZED`
- `INIT_FAILED`

---

### `get_project`

Resolves whether the project has already been initialized.

Payload:
- `projectPath: string` required
- `storageBaseDir: string` expected

Success result:
- `initialized: boolean`
- `project?: ProjectMetadata`

`ProjectMetadata` fields:
- `version: number`
- `projectId: string`
- `name: string`
- `slug: string`
- `hash: string`
- `projectPath: string`
- `projectRootStrategy: string`
- `projectDir: string`
- `dbPath: string`
- `createdAt: string`
- `updatedAt: string`
- `lastOpenedAt?: string`
- `status: string`

Possible errors:
- `INVALID_PAYLOAD`
- `INVALID_PROJECT_PATH`
- `PROJECT_LOOKUP_FAILED`

---

### `project_status`

Returns current project memory status.

Payload:
- `projectPath: string` required
- `storageBaseDir: string` expected

Success result:
- `initialized: boolean`
- `projectId?: string`
- `dbPath?: string`
- `activeMemoryCount: number`
- `trackedSessionCount: number`
- `lastIngestedAt: string`

Possible errors:
- `INVALID_PAYLOAD`
- `INVALID_PROJECT_PATH`
- `PROJECT_STATUS_FAILED`

---

### `ingest_sessions`

Runs session ingestion for the project.

Payload:
- `projectPath: string` required
- `storageBaseDir: string` expected
- `sessionDir?: string`
- `trigger?: string`
- `activeSessionFile?: string`

If `trigger` is omitted, the backend defaults it to `manual`.

Success result:
- `runId: string`
- `trackedSessionsDiscovered: number`
- `sessionFilesProcessed: number`
- `entriesSeen: number`
- `candidatesFound: number`
- `memoriesCreated: number`
- `memoriesUpdated: number`
- `memoriesIgnored: number`
- `lastIngestedAt: string`

Possible errors:
- `INVALID_PAYLOAD`
- `INVALID_PROJECT_PATH`
- `PROJECT_LOOKUP_FAILED`
- `PROJECT_NOT_INITIALIZED`
- `DB_ERROR`
- `INGEST_FAILED`

---

### `rebuild_project_memory`

Clears derived project memory state and re-runs ingestion.

Payload:
- same as `ingest_sessions`

Success result:
- `clearedMemorySources: number`
- `clearedMemoryItems: number`
- `clearedIngestionState: number`
- `clearedIngestionRuns: number`
- `ingest: IngestSessionsResult`

Possible errors:
- `INVALID_PAYLOAD`
- `INVALID_PROJECT_PATH`
- `PROJECT_LOOKUP_FAILED`
- `PROJECT_NOT_INITIALIZED`
- `DB_ERROR`
- `INGEST_FAILED`

---

### `list_memories`

Lists stored memories.

Payload:
- `projectPath: string` required
- `storageBaseDir: string` expected
- `status?: string`
- `limit?: number`

Success result:

```json
{
  "items": [MemoryRow]
}
```

`MemoryRow` fields:
- `memoryId: string`
- `category: string`
- `summary: string`
- `details?: string`
- `status: string`
- `confidence: number`
- `importance: number`
- `updatedAt: string`
- `score?: number`

Possible errors:
- `INVALID_PAYLOAD`
- `INVALID_PROJECT_PATH`
- `PROJECT_LOOKUP_FAILED`
- `PROJECT_NOT_INITIALIZED`
- `DB_ERROR`
- `SEARCH_FAILED`

---

### `search_memories`

Searches structured memories.

Payload:
- `projectPath: string` required
- `storageBaseDir: string` expected
- `query: string` required
- `limit?: number`

Success result:

```json
{
  "items": [MemoryRow]
}
```

Possible errors:
- `INVALID_PAYLOAD`
- `INVALID_PROJECT_PATH`
- `INVALID_QUERY`
- `PROJECT_LOOKUP_FAILED`
- `PROJECT_NOT_INITIALIZED`
- `DB_ERROR`
- `SEARCH_FAILED`

---

### `recall_memories`

Returns the most relevant recall candidates.

Payload:
- `projectPath: string` required
- `storageBaseDir: string` expected
- `limit?: number`

Success result:

```json
{
  "items": [RecallMemoryRow]
}
```

`RecallMemoryRow` extends `MemoryRow` with:
- `recallScore: number`

Possible errors:
- `INVALID_PAYLOAD`
- `INVALID_PROJECT_PATH`
- `PROJECT_LOOKUP_FAILED`
- `PROJECT_NOT_INITIALIZED`
- `DB_ERROR`
- `SEARCH_FAILED`

---

### `search_sessions`

Searches raw tracked session history.

Payload:
- `projectPath: string` required
- `storageBaseDir: string` expected
- `sessionDir?: string`
- `query: string` required
- `limit?: number`

Success result:

```json
{
  "items": [SessionSearchRow]
}
```

`SessionSearchRow` fields:
- `sessionFile: string`
- `entryId?: string`
- `role?: string`
- `excerpt: string`
- `score: number`

Possible errors:
- `INVALID_PAYLOAD`
- `INVALID_PROJECT_PATH`
- `INVALID_QUERY`
- `PROJECT_LOOKUP_FAILED`
- `PROJECT_NOT_INITIALIZED`
- `DB_ERROR`
- `SEARCH_FAILED`

---

### `forget_memory`

Changes a memory item's status.

Payload:
- `projectPath: string` required
- `storageBaseDir: string` expected
- `memoryId: string` required
- `mode?: "suppressed" | "forgotten"`

Behavior:
- invalid or empty `mode` is normalized to `suppressed`

Success result:
- `memoryId: string`
- `status: string`
- `updatedAt: string`

Possible errors:
- `INVALID_PAYLOAD`
- `INVALID_PROJECT_PATH`
- `INVALID_MEMORY_ID`
- `PROJECT_LOOKUP_FAILED`
- `PROJECT_NOT_INITIALIZED`
- `DB_ERROR`
- `MEMORY_NOT_FOUND`

---

### `remember_memory`

Stores an explicit memory item.

Payload:
- `projectPath: string` required
- `storageBaseDir: string` expected
- `text: string` required

Success result:
- `memoryId: string`
- `category: string`
- `summary: string`
- `status: string`
- `confidence: number`
- `importance: number`
- `updatedAt: string`
- `created: boolean`

Possible errors:
- `INVALID_PAYLOAD`
- `INVALID_PROJECT_PATH`
- `INVALID_TEXT`
- `PROJECT_LOOKUP_FAILED`
- `PROJECT_NOT_INITIALIZED`
- `DB_ERROR`

## Error codes

Observed backend/process error codes include:
- `INVALID_REQUEST`
- `INVALID_PAYLOAD`
- `INVALID_PROJECT_PATH`
- `INVALID_QUERY`
- `INVALID_TEXT`
- `INVALID_MEMORY_ID`
- `COMMAND_NOT_IMPLEMENTED`
- `PROJECT_ALREADY_INITIALIZED`
- `PROJECT_LOOKUP_FAILED`
- `PROJECT_STATUS_FAILED`
- `PROJECT_NOT_INITIALIZED`
- `INIT_FAILED`
- `INGEST_FAILED`
- `SEARCH_FAILED`
- `MEMORY_NOT_FOUND`
- `DB_ERROR`

Extension-side wrapper errors may also surface:
- `BACKEND_NOT_FOUND`
- `BACKEND_SPAWN_FAILED`
- `BACKEND_NON_ZERO_EXIT`
- `BACKEND_INVALID_RESPONSE`
- `UNSUPPORTED_PLATFORM`
- `UNSUPPORTED_ARCH`

## Backend resolution

The TypeScript wrapper resolves the backend binary in this order:
1. `PI_MEMORY_BACKEND_PATH`
2. `dist/package/bin/pi-memory-backend`
3. `resources/bin/<platform>-<arch>/<binary>`

## Debug logging

When debug is enabled, the backend writes debug logs to stderr.

Environment variable:

```text
PI_MEMORY_DEBUG=1
```

Accepted truthy values in the backend:
- `1`
- `true`
- `TRUE`
- `yes`
- `on`
