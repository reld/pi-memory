# pi-memory

Pi package for explicit, per-project long-term memory in Pi.

## Overview

`pi-memory` combines:
- a TypeScript Pi extension in `src/extension/`
- a thin Pi entrypoint in `extensions/pi-memory.ts`
- a Go backend in `go/`
- packaged backend binaries in `resources/bin/`

The package reads local Pi session history, extracts durable project memories, stores them in a per-project SQLite database, and recalls relevant memory in later sessions.

## Current status

Core MVP functionality is implemented:
- explicit project initialization
- one SQLite DB per project
- automatic session ingestion
- session-start catch-up ingestion
- session-start memory recall
- turn-end auto-ingest
- manual memory list/search/remember/forget/rebuild commands
- LLM-callable memory tools for on-demand retrieval
- raw session fallback search
- backend JSON-over-stdio integration

The extension keeps automatic session-start recall concise and exposes explicit memory tools for on-demand retrieval instead of injecting recalled memory into every turn's system prompt.

## Commands

- `/pi-memory-init`
- `/pi-memory-status`
- `/pi-memory-config`
- `/pi-memory-config-set <key> <value>`
- `/pi-memory-config-unset <key>`
- `/pi-memory-ingest`
- `/pi-memory-list`
- `/pi-memory-search <query>`
- `/pi-memory-search-sessions <query>`
- `/pi-memory-forget <memoryId>`
- `/pi-memory-remember <text>`
- `/pi-memory-rebuild`

## Runtime configuration

Environment variables:
- `PI_MEMORY_STORAGE_BASE_DIR`
- `PI_MEMORY_BACKEND_PATH`
- `PI_MEMORY_SESSION_DIR`
- `PI_MEMORY_AUTO_INGEST`
- `PI_MEMORY_AUTO_RECALL`
- `PI_MEMORY_RECALL_LIMIT`
- `PI_MEMORY_RAW_SESSION_SEARCH_ENABLED`
- `PI_MEMORY_DEBUG`

Stored global config file:
- `~/.config/pi-memory/config.json`

Effective config precedence:
- explicit command input
- environment variables
- stored global config
- defaults

## Package structure

- `extensions/pi-memory.ts` — thin package entrypoint
- `src/extension/` — Pi extension implementation
- `go/` — Go backend
- `dist/package/bin/` — built backend binary output
- `resources/bin/` — packaged platform binary layout

## Development

Build and validate:

```bash
vp run typecheck
vp run build
```

The Go build script places the backend at:

```text
dist/package/bin/pi-memory-backend
```

If needed, override backend resolution with:

```bash
export PI_MEMORY_BACKEND_PATH=/absolute/path/to/pi-memory-backend
```

## Pi package notes

This repository is intended to be distributed as a Pi package and includes the `pi` manifest in `package.json`.

Install locally in Pi with a path such as:

```bash
pi install /absolute/path/to/pi-memory
```

## Documentation

- `docs/usage.md` — how to use the package day to day
- `docs/extension-api.md` — Pi command/tool/config surface
- `docs/backend-api.md` — JSON-over-stdio backend contract
- `VALIDATION.md` — manual validation findings and current retrieval gaps

See also:
- `AGENTS.md`
- `HANDOFF.md`
- `TODOS.md`
- `THOUGHTS.md`
