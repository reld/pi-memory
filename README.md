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

Development repo:
- `extensions/pi-memory.ts` — thin package entrypoint
- `src/extension/` — Pi extension implementation
- `go/` — Go backend source
- `dist/package/bin/` — local development backend binary output
- `resources/bin/` — packaged platform binary layout

Published package intent:
- include runtime TypeScript extension files
- include packaged backend binaries under `resources/bin/`
- exclude Go backend source from the published package

Source-repo note:
- packaged backend binaries are build artifacts and are not intended to stay committed in the source repo
- local builds may populate `resources/bin/...`, but GitHub Actions is responsible for building them for package publication

## Development

Build and validate:

```bash
vp run typecheck
vp run build
```

The local development build places the backend at:

```text
dist/package/bin/pi-memory-backend
```

Packaged per-platform backend binaries are written to:

```text
resources/bin/<platform>-<arch>/pi-memory-backend
```

Runtime backend resolution order is:
1. `PI_MEMORY_BACKEND_PATH`
2. `resources/bin/<platform>-<arch>/<binary>`
3. `dist/package/bin/<binary>`

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

## Private package publishing direction

Current private distribution target:
- GitHub Packages npm registry
- package name: `@reld/pi-memory`
- packaged backend target: `darwin-arm64`

The release flow is:
1. push source changes
2. create/push a version tag such as `v0.1.0`
3. GitHub Actions validates and publishes the package to GitHub Packages

The published package ships:
- the TypeScript extension runtime files
- the compiled backend binary

It does not ship:
- Go backend source
- other development-only repo files

If your target machine is outside `darwin-arm64`, build a backend there and set `PI_MEMORY_BACKEND_PATH`.

## Installing the private package on another machine

On the target machine, add this copy-paste block to `~/.npmrc`:

```text
@reld:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=YOUR_GITHUB_TOKEN
```

The token should be able to read private packages.

Then install the package in Pi with an explicit version such as:

```bash
pi install npm:@reld/pi-memory@0.1.0
```

For more detailed release/install notes, see:
- `docs/distribution.md`
- `docs/release-checklist.md`

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
