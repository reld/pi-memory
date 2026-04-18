# Usage Guide

This guide explains how to use `pi-memory` in normal day-to-day work.

## What the tool does

`pi-memory` gives Pi a local, explicit, per-project memory layer.

It can:
- initialize a memory database for a project
- ingest local Pi session history
- remember durable facts and preferences
- recall relevant memory in later sessions
- search both structured memory and raw session history
- let you inspect and suppress stored memory

## Typical workflow

### 1. Initialize memory for a project

In the project you want Pi Memory to track:

```text
/pi-memory-init
```

This creates project metadata and a per-project SQLite database.

You only need to do this once per project.

---

### 2. Check status

```text
/pi-memory-status
```

Use this to confirm:
- the project is initialized
- which database is being used
- how many memories exist
- whether ingestion and recall are enabled

---

### 3. Let the extension ingest session history automatically

By default, Pi Memory will:
- ingest on session start to catch up
- ingest after completed assistant turns
- show a small amount of relevant recall on session start

In many cases, you can just use Pi normally and let memory accumulate over time.

---

### 4. Save something explicitly when it matters

If you want to store a durable fact, preference, or instruction directly:

```text
/pi-memory-remember Use VitePlus for TypeScript workflows in this repo.
```

Good things to remember explicitly:
- workflow preferences
- project conventions
- important constraints
- recurring goals
- durable decisions

Examples:

```text
/pi-memory-remember Commit messages should include a short title, blank line, and compact bullet list.
/pi-memory-remember This package ships as a Pi package with a TypeScript extension and Go backend.
/pi-memory-remember Prefer vp over npm in this repository.
```

---

### 5. List and search stored memory

List current active memory:

```text
/pi-memory-list
```

Search structured memory:

```text
/pi-memory-search viteplus
/pi-memory-search commit message
/pi-memory-search backend binary
```

Use structured search first when you want concise, curated memory.

---

### 6. Search raw session history when structured memory is not enough

If you want broader historical evidence or the structured memory search misses something:

```text
/pi-memory-search-sessions packaging
/pi-memory-search-sessions what did we say about retrieval relevance
```

Use this when:
- you want exact-ish historical snippets
- structured memory is too sparse
- you want to inspect conversation evidence directly

---

### 7. Suppress bad or unwanted memory

If a memory item should stop being used:

```text
/pi-memory-forget <memoryId>
```

You can get memory IDs from:
- `/pi-memory-list`
- `/pi-memory-search <query>`

This does not hard-delete the database record in the current UX path.
It marks the memory as suppressed so it no longer participates in normal recall.

---

### 8. Rebuild memory if extraction rules changed or the DB looks wrong

```text
/pi-memory-rebuild
```

Use rebuild when:
- ingestion heuristics changed
- stored memories are noisy or stale
- you want to regenerate derived memory from session history

## Recommended usage patterns

### Best for durable information

Pi Memory works best for information that is likely to matter again later, such as:
- preferences
- conventions
- project facts
- technical constraints
- decisions
- follow-up tasks

### Not ideal for every transient detail

Avoid treating it like a complete transcript replacement.
Use it for durable memory, not every temporary conversational detail.

### Use a layered retrieval approach

Recommended order:
1. let automatic recall help on session start
2. use `/pi-memory-search <query>` for curated memory
3. use `/pi-memory-search-sessions <query>` for broader raw history

## Configuration

Show current effective config:

```text
/pi-memory-config
```

Persist a config value:

```text
/pi-memory-config-set auto-ingest off
/pi-memory-config-set auto-recall off
/pi-memory-config-set recall-limit 8
```

Remove a stored config value:

```text
/pi-memory-config-unset auto-ingest
/pi-memory-config-unset recall-limit
```

Stored config file:

```text
~/.config/pi-memory/config.json
```

Environment variables override stored config.

## Example setup scenarios

### Disable automatic recall but keep ingestion on

```text
/pi-memory-config-set auto-recall off
/pi-memory-config-set auto-ingest on
```

### Increase how many memories are shown on session start

```text
/pi-memory-config-set recall-limit 8
```

### Disable raw session search

```text
/pi-memory-config-set raw-session-search off
```

### Point the extension at a custom backend binary for development

```text
/pi-memory-config-set backend-path /absolute/path/to/pi-memory-backend
```

## How the model should use the tools

The extension also exposes LLM-callable tools.

Expected retrieval order:
1. use `pi_memory_recall` when the user asks what was discussed before or where work left off
2. use `pi_memory_search` for targeted remembered facts, preferences, decisions, tasks, or conventions
3. if structured memory is empty or insufficient for a history question, use `pi_memory_search_sessions`

## Troubleshooting

### “Pi Memory is not initialized for this project”

Run:

```text
/pi-memory-init
```

### “Raw session search is disabled by configuration”

Either enable it:

```text
/pi-memory-config-set raw-session-search on
```

or inspect current config with:

```text
/pi-memory-config
```

### Backend binary cannot be found

Build the project and/or set a backend path:

```text
vp run build
export PI_MEMORY_BACKEND_PATH=/absolute/path/to/pi-memory-backend
```

### Memory looks stale or noisy

Try:
- `/pi-memory-list`
- `/pi-memory-search <query>`
- `/pi-memory-forget <memoryId>` for bad items
- `/pi-memory-rebuild` if you want to regenerate derived memory

## Quick reference

Initialize:

```text
/pi-memory-init
```

Status:

```text
/pi-memory-status
```

Remember:

```text
/pi-memory-remember <text>
```

List:

```text
/pi-memory-list
```

Search memory:

```text
/pi-memory-search <query>
```

Search sessions:

```text
/pi-memory-search-sessions <query>
```

Forget:

```text
/pi-memory-forget <memoryId>
```

Rebuild:

```text
/pi-memory-rebuild
```

Config:

```text
/pi-memory-config
/pi-memory-config-set <key> <value>
/pi-memory-config-unset <key>
```
