# Extension API

This document describes the Pi-facing API exposed by `pi-memory`.

## Scope

The extension API includes:
- slash commands registered in Pi
- LLM-callable tools registered in Pi
- runtime configuration behavior that affects extension behavior

Source of truth:
- `src/extension/index.ts`
- `src/extension/services/config.ts`

## Project context

Most commands and tools operate on the **current Pi working directory** (`ctx.cwd`).
That directory is treated as the current project path and is used to resolve the project-specific memory database.

If a project has not been initialized yet, commands that require a database will fail until:

```text
/pi-memory-init
```

## Commands

### `/pi-memory-init`

Initializes Pi Memory for the current project.

Behavior:
- resolves the current project path from `ctx.cwd`
- resolves the storage base directory from effective runtime config
- creates project metadata and a project SQLite database if needed

Example:

```text
/pi-memory-init
```

Success output includes:
- `project id`
- `project dir`
- `db path`

---

### `/pi-memory-status`

Shows project memory status for the current project.

Example:

```text
/pi-memory-status
```

Output includes:
- `project id`
- `db path`
- `storage base dir`
- `session dir override`
- `active memories`
- `tracked sessions`
- `last ingested at`
- `auto ingest`
- `auto recall`
- `raw session search`

---

### `/pi-memory-config`

Shows the effective runtime configuration.

Example:

```text
/pi-memory-config
```

Output includes:
- config file path
- effective value for each config key
- source for each value: `default`, `file`, or `env`
- precedence reminder: `env > stored config > defaults`

---

### `/pi-memory-config-set <key> <value>`

Persists a runtime config value to the global config file.

Example:

```text
/pi-memory-config-set auto-ingest off
/pi-memory-config-set recall-limit 8
/pi-memory-config-set storage-base-dir /Users/me/.pi-memory
```

Notes:
- environment variables still override stored config values
- values are stored in `~/.config/pi-memory/config.json`

Supported keys:
- `storageBaseDir` / `storage-base-dir`
- `backendPathOverride` / `backend-path`
- `sessionDirOverride` / `session-dir`
- `debug`
- `autoIngest` / `auto-ingest`
- `autoRecall` / `auto-recall`
- `recallLimit` / `recall-limit`
- `rawSessionSearchEnabled` / `raw-session-search`

Accepted boolean values:
- `true`, `false`
- `on`, `off`
- `yes`, `no`
- `1`, `0`

---

### `/pi-memory-config-unset <key>`

Removes a stored config value from the global config file.

Example:

```text
/pi-memory-config-unset backend-path
/pi-memory-config-unset recall-limit
```

After unsetting a key, the effective value may still come from:
- an environment variable
- the built-in default

---

### `/pi-memory-ingest`

Manually ingests tracked session data for the current project.

Example:

```text
/pi-memory-ingest
```

Output includes:
- `run id`
- `tracked sessions discovered`
- `session files processed`
- `entries seen`
- `candidates found`
- `memories created`
- `memories updated`
- `memories ignored`
- `last ingested at`

---

### `/pi-memory-list`

Lists active stored memories for the current project.

Example:

```text
/pi-memory-list
```

---

### `/pi-memory-search <query>`

Searches structured stored memories.

Example:

```text
/pi-memory-search viteplus
/pi-memory-search commit message format
```

---

### `/pi-memory-search-sessions <query>`

Searches raw tracked session history for the current project.

Example:

```text
/pi-memory-search-sessions packaging
/pi-memory-search-sessions remember arepas
```

Notes:
- this is a fallback/safety-net search layer
- it is disabled when `rawSessionSearchEnabled` is off

---

### `/pi-memory-forget <memoryId>`

Suppresses a stored memory item.

Example:

```text
/pi-memory-forget mem_123
```

Current behavior:
- sets memory status to `suppressed`

---

### `/pi-memory-remember <text>`

Stores an explicit memory item for the current project.

Example:

```text
/pi-memory-remember Use VitePlus for TS workflows in this repo.
```

Output includes:
- `memory id`
- `category`
- `summary`
- `status`
- `confidence`
- `importance`
- whether it was newly created or updated

---

### `/pi-memory-rebuild`

Clears derived memory state and re-ingests tracked sessions.

Example:

```text
/pi-memory-rebuild
```

Output includes:
- cleared counts for memory and ingestion state
- nested ingest result summary

## LLM-callable tools

These tools are registered for model use inside Pi.

### `pi_memory_recall`

Purpose:
- recall the most relevant stored project memories

Parameters:
- `limit?: number` with range `1..20`

Typical use:
- user asks what was discussed before
- user asks where work left off
- user asks what should be remembered

---

### `pi_memory_search`

Purpose:
- search structured stored project memories

Parameters:
- `query: string`
- `limit?: number` with range `1..20`

Typical use:
- user asks for a remembered preference, decision, fact, task, or convention

Important behavior:
- if the result is empty or insufficient for a history question, the model should use `pi_memory_search_sessions`

---

### `pi_memory_search_sessions`

Purpose:
- search raw tracked session history as a fallback

Parameters:
- `query: string`
- `limit?: number` with range `1..20`

Typical use:
- structured memory lookup was empty or not enough
- user is asking about prior conversation details

Constraint:
- disabled when `rawSessionSearchEnabled` is false

## Lifecycle behavior

### `session_start`

On session start, the extension:
1. resolves the current project
2. checks whether the project is initialized
3. optionally runs catch-up ingestion if `autoIngest` is enabled
4. optionally shows concise recall results if `autoRecall` is enabled

### `turn_end`

On completed assistant turns, the extension:
1. checks whether the current project is initialized
2. runs automatic incremental ingestion if `autoIngest` is enabled

## Runtime configuration

Effective config is resolved from:
1. explicit command input
2. environment variables
3. stored global config
4. defaults

Config file:

```text
~/.config/pi-memory/config.json
```

Supported fields:
- `storageBaseDir: string`
- `backendPathOverride?: string`
- `sessionDirOverride?: string`
- `debug: boolean`
- `autoIngest: boolean`
- `autoRecall: boolean`
- `recallLimit: number`
- `rawSessionSearchEnabled: boolean`

Environment variables:
- `PI_MEMORY_STORAGE_BASE_DIR`
- `PI_MEMORY_BACKEND_PATH`
- `PI_MEMORY_SESSION_DIR`
- `PI_MEMORY_AUTO_INGEST`
- `PI_MEMORY_AUTO_RECALL`
- `PI_MEMORY_RECALL_LIMIT`
- `PI_MEMORY_RAW_SESSION_SEARCH_ENABLED`
- `PI_MEMORY_DEBUG`

## Error behavior

Common extension-level failures:
- project not initialized
- backend binary not found
- invalid command usage
- raw session search disabled by config
- backend command errors surfaced through `BackendError`

The extension reports errors through Pi notifications.

Current notable error mappings:
- `PROJECT_NOT_INITIALIZED` → suggests `/pi-memory-init`
- `PROJECT_ALREADY_INITIALIZED` → suggests `/pi-memory-status`
- `BACKEND_NOT_FOUND` → suggests `vp run build` or `PI_MEMORY_BACKEND_PATH`
- `INVALID_QUERY` → asks for a non-empty query
- `INVALID_TEXT` → asks for non-empty remember text
- `INVALID_MEMORY_ID` / `MEMORY_NOT_FOUND` → suggests `/pi-memory-list` or `/pi-memory-search`
