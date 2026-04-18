# THOUGHTS

## Pi sessions and local persistence

Facts we confirmed:

- Pi sessions are typically persisted locally as JSONL files.
- The default session directory can be configured via `sessionDir` and is commonly under `~/.pi/agent/sessions/...`.
- Session persistence can be disabled with `--no-session`.
- SDK usage can also avoid persistence by using `SessionManager.inMemory()`.
- Stored session data can include:
  - user messages
  - assistant messages
  - tool results
  - bash execution records
  - custom extension entries/messages
  - branch and summary metadata
- Stored locally does **not** mean Pi automatically remembers all past sessions in the current chat context.
- However, if session files are available on disk and tools/extensions are allowed to read them, they can be inspected like other local files.
- This means prior sessions can be searched, summarized, indexed, and used as a basis for long-term memory.

## Extension idea: long-term memory via sessions

Goal:

Build a Pi extension that uses locally stored session history to create memory across sessions.

What this likely means:

- read prior session files
- extract useful facts/preferences/tasks/decisions
- store or index memory in a structured way
- surface relevant memory in future sessions
- keep the behavior explicit and controllable

Important design principle:

- memory should be intentional, inspectable, and easy to limit or disable
- the extension should avoid pretending to "know everything" automatically
- ideally, it should only recall relevant information when helpful

## Architecture direction

Current preference:

- Use a **hybrid memory model**.
- Use **per-project memory** as a first-class concept.
- Use **one SQLite database per project**.
- Start with **TypeScript + Go from the beginning**.
- Databases should be created during setup/initialization, with the user choosing the base storage location.

Why SQLite makes sense:

- simple local file-based database
- excellent fit for one-DB-per-project state
- supports indexing and filtering well
- easy to inspect and back up
- can later support FTS or vector-style side tables if needed

Possible implementation split:

- a **Pi extension in TypeScript** for integration with Pi
- a **Go binary** for database operations, indexing, and search
- package everything together as a Pi package

Current storage direction:

- user selects a base directory for memory storage
- each project gets its own directory and database
- example path: `~/.pi-memory/[projectname-slug-with-hash]/memory.db`
- project directory naming should be deterministic and collision-resistant
- setup should make the storage path explicit and reviewable by the user
- after installation/setup, the base directory should also contain a global `projects.json` registry
- each project directory should contain:
  - `project.json` as the per-project metadata source of truth
  - `memory.db` as the project's SQLite memory store
- `projects.json` should be a lightweight global registry of known projects and pointers to their `project.json` files
- project initialization should be explicit via a Pi command such as `/pi-memory-init`

Reasoning:

- Pi extensions themselves are TypeScript modules
- but the extension can call external programs, so a Go helper/backend is a valid architecture
- this keeps Pi-specific integration in TypeScript while moving heavier storage/search logic into a compiled binary
- starting with the final split from day one avoids a future rewrite from TS-only internals to Go-backed internals

Important caveats:

- if we ship a Go binary, packaging and installation become more complex
- we need a strategy for building or distributing binaries for target platforms
- we should keep memory storage transparent and inspectable by the user
- setup UX matters because DB creation is explicit and location-sensitive
- we should clearly define which metadata is authoritative in `project.json` vs mirrored/indexed in `projects.json`

## Product definition

Pi Memory is a Pi extension and companion Go backend that gives Pi explicit, per-project long-term memory across sessions by reading local Pi session history, extracting durable memories such as preferences, decisions, facts, tasks, and conventions, storing them in a project-specific SQLite database, and surfacing only relevant, inspectable, user-controllable memory in future sessions through clear commands, setup, and review flows.

## MVP scope

First version should focus on a narrow, explicit, useful memory loop.

Included in MVP:

- explicit project setup via `/pi-memory-init`
- creation of the global `projects.json` registry if missing
- creation of a per-project folder containing `project.json` and `memory.db`
- one SQLite database per initialized project
- TypeScript extension + Go binary integration from day one
- project resolution based on the current workspace/project path
- automatic incremental ingestion of Pi session history into candidate memories
- manual explicit memory capture for commands like “remember this”
- storage of structured memory items with source traceability
- manual listing/searching of stored memories
- manual forgetting/deleting/suppressing of memories
- session-start recall of a small number of relevant memories
- clear user visibility into what is stored and why

Memory types in MVP:

- preferences
- project facts
- decisions
- tasks / follow-ups
- constraints / conventions

Not included in MVP:

- global cross-project memory
- fully opaque or uncontrollable memory creation
- embeddings/vector search as a requirement
- advanced semantic ranking beyond a practical first-pass approach
- multi-user or remote/shared memory
- automatic recovery of moved/renamed projects beyond basic relink support
- complex UI beyond commands and simple review flows

Suggested MVP commands:

- `/pi-memory-init`
- `/pi-memory-ingest`
- `/pi-memory-search`
- `/pi-memory-list`
- `/pi-memory-forget`
- `/pi-memory-status`

Success criteria for MVP:

- a user can initialize memory for a project explicitly
- the extension creates and reopens the correct per-project DB reliably
- session history can be ingested automatically into useful, inspectable memory items
- relevant memories can be surfaced in later sessions without being noisy
- users can inspect and remove memories they do not want kept

## Ingestion strategy

Core decision:

- ingestion should be automatic so the user can stay in flow
- manual explicit memory capture should still exist for cases like “remember this”

Recommended behavior:

- automatically ingest incrementally after each completed assistant turn
- only process new session content that has not yet been ingested
- trigger catch-up ingestion on session start if backlog exists
- keep a manual ingestion command for debugging or recovery

Important principle:

- Pi session files are the durable raw source of truth
- the memory database is derived structured state
- not everything recorded in the session should become a durable memory item

Crash/recovery model:

- if Pi persists the session, raw conversation/tool history survives crashes
- the memory system should be able to resume by ingesting missing session entries later
- this requires ingestion bookkeeping and source traceability

Likely ingestion requirements:

- track which session files/entries have already been processed
- support incremental ingestion via checkpoints/cursors
- deduplicate aggressively to avoid repeated low-value memories
- distinguish explicit user memory requests from passive automatic extraction
- keep memory creation selective even when ingestion is automatic

## Ingestion decision: low-token algorithmic backend first

Core decision:

- ingestion should use as few model tokens as possible
- the default ingestion path should be algorithmic and handled primarily by the Go backend
- model-assisted extraction should be optional and not required for the MVP core loop

What ingestion means:

- read new/unprocessed Pi session data
- extract candidate memories from the session stream
- score/filter/deduplicate candidates
- persist only durable, useful memory items into the per-project database

What should decide whether something becomes memory:

- primarily deterministic rules and scoring in Go
- not the LLM on every turn

Recommended layered approach:

1. algorithmic extraction and scoring in Go
2. optional model-assisted refinement later, only for special cases

Good memory characteristics:

- durable beyond the current turn
- useful in future sessions
- specific enough to act on
- non-duplicative
- relevant to the current project

Examples of things the algorithmic backend can detect:

- explicit user phrases like “remember this”
- preferences such as “I prefer...” or “please use...”
- decisions such as “we decided...”
- tasks/follow-ups such as “next step...” or “we still need to...”
- constraints/conventions such as “never modify...” or “this project uses...”

Design implication:

- the Go backend should own candidate extraction, scoring, deduplication, and persistence
- the TypeScript extension should mainly trigger ingestion and present results in Pi

## TypeScript vs Go responsibilities

### TypeScript extension responsibilities

The TypeScript side should own everything that is Pi-facing and interaction-oriented.

Responsibilities:

- register Pi commands such as `/pi-memory-init`, `/pi-memory-ingest`, `/pi-memory-search`, `/pi-memory-list`, `/pi-memory-forget`, and `/pi-memory-status`
- integrate with Pi session lifecycle events such as session start
- resolve the current project/workspace context
- determine whether the current project has already been initialized
- guide the user through setup and initialization flows
- display memory results, status messages, prompts, confirmations, and errors inside Pi
- invoke the Go binary for storage, ingestion, and retrieval operations
- translate Pi/session concepts into requests to the Go backend
- format retrieved memories for injection or display in Pi
- enforce user-facing safety and control flows before destructive actions

The TypeScript side should not own the core persistence/search logic if that logic is already delegated to Go.

### Go backend responsibilities

The Go side should own the durable backend and memory engine.

Responsibilities:

- create and manage the global storage home
- create and update `projects.json`
- create project directories
- create and update per-project `project.json`
- initialize and migrate each project's `memory.db`
- implement SQLite access
- implement ingestion bookkeeping
- parse/process ingestion inputs provided by the TypeScript side
- read session-derived inputs and perform algorithmic candidate extraction
- store memory items and source references
- implement listing, lookup, filtering, and search over stored memory
- implement deduplication logic
- implement scoring/relevance helpers as needed for retrieval
- return structured machine-readable results to the TypeScript extension
- own consistency of on-disk state

The Go side should not be responsible for Pi UI decisions or Pi-specific UX behavior.

### Boundary principle

- TypeScript owns **Pi integration and UX**.
- Go owns **state, storage, indexing, and retrieval**.
- Keep business logic close to the storage layer when it affects correctness and consistency.
- Keep interaction logic close to Pi when it affects user experience.

### Practical rule of thumb

If a feature answers **"what should the user see or be asked inside Pi?"**, it belongs primarily in TypeScript.

If a feature answers **"what should be stored, indexed, migrated, searched, or resolved on disk/in SQLite?"**, it belongs primarily in Go.

## TS ↔ Go communication model

Recommended approach for MVP:

- use **JSON-over-stdio** between the TypeScript extension and the Go binary
- the TypeScript side launches the Go binary as a subprocess when needed
- the TypeScript side sends a structured JSON request
- the Go binary returns a structured JSON response
- keep the protocol request/response oriented, not long-running service oriented, for the MVP

Why this is a good fit:

- simple to reason about
- no local server lifecycle to manage
- no port conflicts
- good fit for explicit commands like init, ingest, search, list, and forget
- works well with packaging a local binary alongside a Pi extension

### Recommended protocol shape

Each invocation should follow a simple envelope such as:

Request:

```json
{
  "version": 1,
  "command": "init_project",
  "payload": {
    "projectPath": "/absolute/path",
    "storageBaseDir": "/Users/reld/.pi-memory"
  }
}
```

Response:

```json
{
  "ok": true,
  "result": {
    "projectId": "pi-memory-1a2b3c4d",
    "projectDir": "/Users/reld/.pi-memory/pi-memory-1a2b3c4d",
    "projectFile": "/Users/reld/.pi-memory/pi-memory-1a2b3c4d/project.json",
    "dbPath": "/Users/reld/.pi-memory/pi-memory-1a2b3c4d/memory.db"
  }
}
```

Error response:

```json
{
  "ok": false,
  "error": {
    "code": "PROJECT_ALREADY_INITIALIZED",
    "message": "Project already initialized",
    "details": {}
  }
}
```

### Command model

Likely backend commands for MVP:

- `init_project`
- `get_project`
- `list_projects`
- `ingest_sessions`
- `list_memories`
- `search_memories`
- `forget_memory`
- `suppress_memory`
- `project_status`

### Process model

For MVP, prefer:

- **one-shot subprocess calls** per operation
- no daemon mode required
- no socket/RPC server required

This keeps the first version simple and robust.

If later needed, we can evolve to:

- persistent daemon mode
- local RPC
- streaming progress updates

### Output discipline

- stdout should be reserved for machine-readable JSON responses
- stderr should be used for logs/debug information
- exit code should indicate transport/process failure
- domain failures should still return structured JSON error payloads when possible

### TypeScript wrapper behavior

The TypeScript extension should have a small wrapper that:

- locates the binary
- launches it
- sends request JSON
- parses response JSON
- maps backend error codes to Pi-friendly messages
- handles timeouts and process failures

### Design principle

Use a protocol that is:

- versioned
- explicit
- deterministic
- easy to test from the command line
- independent from Pi internals

## Project identity and DB path rules

### Core decision

A project's memory identity should be based primarily on its **canonical project path at initialization time**, with explicit metadata persisted so it can be reopened reliably later.

### Project root resolution

For MVP, resolve the project root as:

1. git repository root, if the current workspace is inside a git repo
2. otherwise, the current working directory / Pi project directory

The resolved path should then be canonicalized:

- absolute path
- normalized path
- symlinks resolved where practical

This canonical project root becomes the main identity input at initialization time.

### Why this approach

- simple and deterministic
- matches how users think about projects
- works for both git and non-git projects
- good enough for one-project-one-DB setup
- easy to explain during `/pi-memory-init`

### Stable stored identity

At init time, we should persist in `project.json`:

- `projectId`
- `name`
- `slug`
- `hash`
- `projectPath` (canonical path at init time)
- `createdAt`
- `updatedAt`
- `dbPath`

This means the live identity rule is:

- use current resolved canonical project path to look up an existing initialized project in `projects.json`
- once found, trust that project's `project.json` as the source of truth

### Project naming

Use two pieces:

- **slug** = sanitized human-readable project name, usually basename of project root
- **hash** = short deterministic hash derived from canonical project root path

Directory name format:

- `[slug]-[hash]`

Example:

- project root: `/Users/reld/Code/Playground/PI/pi-memory`
- slug: `pi-memory`
- hash: `1a2b3c4d`
- directory: `~/.pi-memory/pi-memory-1a2b3c4d/`

### Hash rules

For MVP, the hash should:

- be deterministic
- be based on the canonical project root path
- be short but collision-resistant enough for local use
- be stable across sessions as long as the project path is stable

A short hex prefix of a stable hash of the canonical path is sufficient for MVP.

### DB path rule

Given:

- base storage dir chosen by user, e.g. `~/.pi-memory`
- project dir name = `[slug]-[hash]`

Then:

- project dir = `[baseStorageDir]/[slug]-[hash]/`
- db path = `[projectDir]/memory.db`
- project metadata path = `[projectDir]/project.json`

### Global registry lookup rule

`projects.json` should store a lightweight list of known projects keyed by their resolved project path and project id.

At startup / command execution:

1. resolve current canonical project root
2. search `projects.json` for matching `projectPath`
3. if found, open the referenced `project.json`
4. if not found, treat project as uninitialized and offer `/pi-memory-init`

### Move / rename behavior

For MVP, moving or renaming a project changes the canonical path, so automatic identity matching may fail.

That is acceptable initially.

Expected behavior:

- if no path match is found, the extension treats the project as uninitialized
- later we can support relinking by comparing metadata and letting the user reconnect the moved project to the existing memory directory

### Important principle

For MVP, project identity is **path-based with explicit persisted metadata**, not hidden magic.

That gives us:

- predictable DB resolution
- simple implementation
- user-understandable behavior
- room to add relinking later

## `projects.json` shape

Purpose:

- lightweight global registry of initialized Pi Memory projects
- quick lookup from current project path to project metadata
- not the primary source of truth for full project metadata

Authority rule:

- `projects.json` is the global index/registry
- `project.json` inside each project directory is the source of truth for that specific project

Recommended MVP shape:

```json
{
  "version": 1,
  "baseStorageDir": "/Users/reld/.pi-memory",
  "projects": [
    {
      "projectId": "pi-memory-1a2b3c4d",
      "name": "pi-memory",
      "slug": "pi-memory",
      "hash": "1a2b3c4d",
      "projectPath": "/Users/reld/Code/Playground/PI/pi-memory",
      "projectDir": "/Users/reld/.pi-memory/pi-memory-1a2b3c4d",
      "projectFile": "/Users/reld/.pi-memory/pi-memory-1a2b3c4d/project.json",
      "dbPath": "/Users/reld/.pi-memory/pi-memory-1a2b3c4d/memory.db",
      "createdAt": "2026-04-18T12:00:00Z",
      "updatedAt": "2026-04-18T12:00:00Z"
    }
  ]
}
```

Minimal required fields per project entry for MVP:

- `projectId`
- `projectPath`
- `projectFile`

Recommended additional convenience fields:

- `name`
- `slug`
- `hash`
- `projectDir`
- `dbPath`
- `createdAt`
- `updatedAt`

Design notes:

- keep this file easy to inspect and repair by hand
- entries should be unique by `projectId`
- `projectPath` should be unique for active linked projects
- if a referenced `project.json` is missing, the registry entry is stale and should be repairable

## `project.json` shape

Purpose:

- authoritative metadata for one initialized project
- defines how that project's memory storage is linked to a real project on disk
- used to reopen the correct memory DB and support future relinking/repair flows

Recommended MVP shape:

```json
{
  "version": 1,
  "projectId": "pi-memory-1a2b3c4d",
  "name": "pi-memory",
  "slug": "pi-memory",
  "hash": "1a2b3c4d",
  "projectPath": "/Users/reld/Code/Playground/PI/pi-memory",
  "projectRootStrategy": "git-root-or-cwd",
  "projectDir": "/Users/reld/.pi-memory/pi-memory-1a2b3c4d",
  "dbPath": "/Users/reld/.pi-memory/pi-memory-1a2b3c4d/memory.db",
  "createdAt": "2026-04-18T12:00:00Z",
  "updatedAt": "2026-04-18T12:00:00Z",
  "lastOpenedAt": "2026-04-18T12:00:00Z",
  "status": "active"
}
```

Required fields for MVP:

- `version`
- `projectId`
- `name`
- `slug`
- `hash`
- `projectPath`
- `projectDir`
- `dbPath`
- `createdAt`
- `updatedAt`

Recommended optional fields:

- `projectRootStrategy`
- `lastOpenedAt`
- `status`

Possible future fields:

- `notes`
- `aliases`
- `previousProjectPaths`
- `relinkedAt`
- `projectFingerprint`
- `settings`

### Field semantics

- `projectId`: stable identifier for this memory project
- `name`: human-readable project name
- `slug`: sanitized name used in directory naming
- `hash`: short deterministic hash derived from canonical path
- `projectPath`: canonical project root path at initialization time
- `projectDir`: directory containing `project.json` and `memory.db`
- `dbPath`: SQLite DB path for this project
- `createdAt`: initialization time
- `updatedAt`: last metadata update time
- `lastOpenedAt`: last time this project memory was used
- `status`: active/suppressed/relinked/deleted-like lifecycle state if needed later

### Design principle

- `project.json` should be sufficient to understand and recover one project's memory setup even if `projects.json` becomes stale
- `projects.json` should be sufficient to discover known projects quickly without scanning every project directory

## What a memory item is

A memory item is a durable, project-relevant piece of information extracted from Pi session history or explicitly saved by the user because it is likely to be useful again in future sessions.

A memory item is **not** just a raw chat message copy.
It is a structured record derived from session data.

### A good memory item should be

- durable beyond the current turn
- useful in future sessions
- specific enough to act on
- relevant to the current project
- traceable back to its source
- deduplicatable against similar existing memories

### Memory item categories for MVP

- `preference`
  - user preferences about how Pi should work in this project
  - example: "Prefer concise answers"
- `fact`
  - stable project facts
  - example: "This project is a Pi package with a TS extension and a Go backend"
- `decision`
  - decisions made during planning or implementation
  - example: "Use one SQLite DB per project"
- `task`
  - unfinished work, next steps, or follow-ups worth recalling later
  - example: "Need to define the SQLite schema"
- `constraint`
  - rules, limits, conventions, or things to avoid
  - example: "Keep token usage low; default ingestion should be algorithmic"

### Suggested MVP fields for a memory item

Conceptually, each memory item should have:

- `memoryId`
- `projectId`
- `category`
- `summary`
- `details` or `body`
- `status`
- `confidence`
- `importance`
- `sourceType`
- `createdAt`
- `updatedAt`

### Field meaning

- `memoryId`: stable identifier for the memory row
- `projectId`: owning project
- `category`: preference/fact/decision/task/constraint
- `summary`: short human-readable memory text
- `details` or `body`: optional richer description
- `status`: active, suppressed, forgotten, deleted-like lifecycle state
- `confidence`: how confident the system is that this is a valid memory
- `importance`: how useful it is likely to be in future recall
- `sourceType`: whether it came from automatic extraction or explicit user intent
- `createdAt`: when first stored
- `updatedAt`: when last changed

### Source types for MVP

- `auto`
- `explicit_user`
- possibly later: `imported`, `merged`, `edited`

### Example memory item

```json
{
  "memoryId": "mem_01",
  "projectId": "pi-memory-1a2b3c4d",
  "category": "decision",
  "summary": "Use one SQLite database per project.",
  "details": "Project memory is stored in one project-specific SQLite DB created during setup.",
  "status": "active",
  "confidence": 0.96,
  "importance": 0.9,
  "sourceType": "explicit_user",
  "createdAt": "2026-04-18T12:00:00Z",
  "updatedAt": "2026-04-18T12:00:00Z"
}
```

### Important distinction

A memory item should capture a **normalized useful fact**, not a raw transcript fragment.

Bad memory item:
- "User said okay let's do that"

Good memory item:
- "User prefers a TS extension plus Go backend from the beginning"

## SQLite schema direction

The per-project SQLite database should stay small, explicit, and inspectable.
For MVP, it should focus on three core concerns:

- stored memory items
- traceability back to session sources
- ingestion bookkeeping

### Core tables for MVP

#### 0. `tracked_sessions`

Explicit registry of session files known to belong to this project.

Why add it:

- clearer project-scoped session tracking
- better debugging/status reporting
- easier raw session fallback search
- cleaner separation between session discovery and ingestion progress

Suggested columns:

- `session_file` TEXT PRIMARY KEY
- `project_id` TEXT NOT NULL
- `session_id` TEXT NULL
- `session_name` TEXT NULL
- `first_seen_at` TEXT NOT NULL
- `last_seen_at` TEXT NOT NULL
- `last_ingested_at` TEXT NULL
- `status` TEXT NOT NULL

Purpose:

- record that a given session file belongs to this project
- track whether it is active/missing/stale if needed
- support reporting and later repair/relink flows

Suggested MVP status values:

- `active`
- `missing`
- `stale`

Notes:

- `session_file` is the canonical absolute path identity key for MVP
- `session_id` can be filled when recoverable from Pi state/content, but is optional in MVP
- `session_name` is optional convenience/debug metadata

#### 1. `memory_items`

Primary table containing normalized memories.

Suggested columns:

- `memory_id` TEXT PRIMARY KEY
- `project_id` TEXT NOT NULL
- `category` TEXT NOT NULL
- `summary` TEXT NOT NULL
- `details` TEXT NULL
- `status` TEXT NOT NULL
- `source_type` TEXT NOT NULL
- `confidence` REAL NOT NULL
- `importance` REAL NOT NULL
- `created_at` TEXT NOT NULL
- `updated_at` TEXT NOT NULL

Purpose:

- one row per durable memory item
- stores the canonical memory text and lifecycle state

Notes:

- `status` can start with values like `active`, `suppressed`, `forgotten`
- `category` is one of the MVP categories
- `details` can store richer text when needed

#### 2. `memory_sources`

Links memory items back to the session evidence they came from.

Suggested columns:

- `id` INTEGER PRIMARY KEY AUTOINCREMENT
- `memory_id` TEXT NOT NULL
- `session_file` TEXT NOT NULL
- `entry_id` TEXT NULL
- `entry_role` TEXT NULL
- `excerpt` TEXT NULL
- `created_at` TEXT NOT NULL

Purpose:

- source traceability
- explainability
- debugging
- future re-ingestion or repair support

Notes:

- one memory item may have multiple source rows
- `excerpt` should be small and useful, not a full transcript dump

#### 3. `ingestion_runs`

Tracks ingestion operations and their outcomes.

Suggested columns:

- `run_id` TEXT PRIMARY KEY
- `started_at` TEXT NOT NULL
- `finished_at` TEXT NULL
- `status` TEXT NOT NULL
- `trigger` TEXT NOT NULL
- `session_file` TEXT NULL
- `entries_seen` INTEGER NOT NULL DEFAULT 0
- `candidates_found` INTEGER NOT NULL DEFAULT 0
- `memories_created` INTEGER NOT NULL DEFAULT 0
- `memories_updated` INTEGER NOT NULL DEFAULT 0
- `memories_skipped` INTEGER NOT NULL DEFAULT 0
- `error_message` TEXT NULL

Purpose:

- debugging
- crash recovery visibility
- ingestion observability

Typical trigger values:

- `auto_turn`
- `session_start_catchup`
- `manual`
- `explicit_user`

#### 4. `ingestion_state`

Tracks incremental progress so we only process new session data.

Suggested columns:

- `session_file` TEXT PRIMARY KEY
- `last_entry_id` TEXT NULL
- `last_entry_timestamp` TEXT NULL
- `last_ingested_at` TEXT NOT NULL
- `last_run_id` TEXT NULL

Purpose:

- incremental ingestion
- catch-up after restart/crash
- avoid reprocessing the whole session every time

Relationship to `tracked_sessions`:

- `tracked_sessions` answers: which session files belong to this project?
- `ingestion_state` answers: how far have we processed each session file?

### Optional MVP-or-soon-after tables

#### `memory_tags`

- `memory_id`
- `tag`

Useful if we want lightweight tagging/filtering.
Could wait until after the first working version.

### Indexes to add early

Recommended indexes:

- on `memory_items(category)`
- on `memory_items(status)`
- on `memory_items(updated_at)`
- on `memory_items(source_type)`
- on `memory_sources(memory_id)`
- on `memory_sources(session_file)`
- on `ingestion_state(last_ingested_at)` if useful

### Schema principle

Keep the schema normalized enough for traceability, but not overengineered.
The first version should optimize for:

- easy debugging
- easy manual inspection
- reliable ingestion bookkeeping
- simple retrieval/filtering

## Ingestion and scoring rules

The ingestion pipeline should be deterministic, low-token, and selective.
Its job is not to store everything, but to convert new session data into a small number of durable, useful memories.

### Pipeline stages

#### 1. Read session delta

For each relevant session file:

- read only entries not yet covered by `ingestion_state`
- process entries in order
- work from the session file as the raw source of truth

#### 2. Extract candidate signals

The backend should inspect new entries and look for candidate memory signals.

Strong signal types for MVP:

- explicit user memory intent
  - e.g. "remember this", "please remember", "note this"
- preference statements
  - e.g. "I prefer...", "please use...", "don't use..."
- decision statements
  - e.g. "we decided...", "let's go with...", "we will use..."
- constraint/convention statements
  - e.g. "always...", "never...", "must...", "this project uses..."
- task/follow-up statements
  - e.g. "next we should...", "still need to...", "todo"
- repeated important facts
  - a fact mentioned multiple times across turns

Potential source entries:

- user messages
- assistant messages
- selected custom/tool-related entries only if they contain durable project information

### Candidate normalization

Candidates should be normalized into a structured form before scoring.

Normalized candidate fields:

- `category`
- `summary`
- `details`
- `source_type`
- `source_refs`
- `signal_flags`

Examples of signal flags:

- `explicit_memory_request`
- `contains_preference_pattern`
- `contains_decision_pattern`
- `contains_constraint_pattern`
- `contains_task_pattern`
- `repeated_signal`
- `assistant_only_statement`
- `low_specificity`

### Scoring model

Use a deterministic weighted score from 0.0 to 1.0.

Two related values should be produced:

- `confidence`: how likely it is that the extracted memory is valid
- `importance`: how useful it is likely to be in future recall

#### Suggested confidence boosters

Increase confidence when:

- explicit memory intent is present
- the statement is user-authored
- the wording strongly matches a known category pattern
- the same fact/preference/decision appears repeatedly
- the candidate is specific and concrete
- the candidate maps cleanly to one category

#### Suggested confidence penalties

Decrease confidence when:

- the text is vague
- the statement is obviously temporary/transient
- the candidate is assistant-only and not grounded by user intent or repeated evidence
- the candidate is too similar to an existing memory with no meaningful new information
- the text is procedural chatter rather than durable knowledge

#### Suggested importance boosters

Increase importance when:

- the memory is a decision, constraint, or strong user preference
- it affects future implementation behavior
- it is likely to prevent future mistakes or repeated clarification
- the user explicitly asked to remember it

#### Suggested importance penalties

Decrease importance when:

- the item is a one-off temporary task
- it is too generic to help later
- it is local to a single moment and unlikely to recur

### Decision thresholds

For MVP, each candidate should end in one of three outcomes:

- **save**
- **suppress**
- **ignore**

Suggested rough policy:

- high confidence + medium/high importance → save
- medium confidence or medium importance → suppress or skip depending on novelty
- low confidence + low importance → ignore

Practical interpretation:

- `save`: create or update a memory item
- `suppress`: do not surface automatically, but optionally keep as a low-priority/internal candidate later if we add that concept
- `ignore`: drop it entirely

For strict MVP simplicity, we can initially implement only:

- save
- ignore

And map `suppressed` to a later enhancement or to memory item status only when explicitly needed.

### Deduplication rules

Before inserting a new memory:

- compare against existing active memories in the same project
- compare by normalized summary and category first
- then use simple text similarity / normalization rules
- if essentially identical, update metadata instead of creating a duplicate

Possible outcomes:

- exact duplicate → refresh/update existing memory
- same idea with better specificity → update existing memory
- meaningfully new information → create new memory

### Explicit user memory rule

If the user explicitly asks Pi to remember something:

- assign very high confidence
- assign high importance by default
- prefer saving unless the extracted memory is empty or malformed
- mark `source_type = explicit_user`

This is the strongest ingestion signal in MVP.

### Assistant-only memory rule

Assistant-only content should be treated cautiously.

Do not save assistant-only statements unless at least one is true:

- it reflects an explicit user-approved decision
- it is repeated and consistent with user/project context
- it is clearly derived from a durable project fact

This avoids Pi inventing memory from its own speculative wording.

### What should usually be ignored

Ignore content like:

- transient procedural chatter
- file-reading / tool-execution narration
- one-off greetings or acknowledgements
- vague statements with no reusable value
- short tactical steps that will not matter later

### Suggested initial category heuristics

- `preference`: phrases like "I prefer", "please use", "don't do X"
- `decision`: phrases like "we decided", "let's go with", "we will use"
- `constraint`: phrases like "must", "never", "always", "keep X low"
- `task`: phrases like "next", "todo", "still need to"
- `fact`: stable project descriptions that do not fit the above but appear concrete and reusable

### Updating an existing memory

When a new signal matches an existing memory:

- update `updated_at`
- optionally raise `confidence`
- optionally raise `importance` if repeated or explicitly reaffirmed
- append new evidence into `memory_sources`

### Safety principle

The system should bias toward **missing a weak memory** rather than **polluting the database with noise**.

A smaller, cleaner memory set is better than an overfull noisy one.

## Exact scoring rubric for MVP

The first scoring system should be simple, deterministic, and easy to tune.
It does not need to be perfect; it needs to be understandable.

### Scoring outputs

Each candidate gets:

- `confidence_score` in the range `0.0 - 1.0`
- `importance_score` in the range `0.0 - 1.0`
- final action: `save`, `ignore`

For MVP, keep the action logic simple and deterministic.

### Base scores

Start every candidate with:

- `confidence = 0.30`
- `importance = 0.30`

Then apply boosters and penalties.
Clamp both values to `0.0 - 1.0` at the end.

## Confidence scoring rules

### Strong boosters

- `+0.50` if `explicit_memory_request`
- `+0.25` if the signal comes from a **user** message
- `+0.20` if it strongly matches a category pattern
- `+0.15` if repeated or reaffirmed in later entries
- `+0.15` if the statement is concrete/specific
- `+0.10` if it maps cleanly to exactly one category

### Penalties

- `-0.30` if clearly temporary/transient
- `-0.25` if low specificity / vague wording
- `-0.20` if assistant-only with no user confirmation
- `-0.20` if mostly procedural/tool narration
- `-0.15` if very similar to an existing memory with no new detail
- `-0.10` if conflicting signals exist nearby and the system cannot resolve them

## Importance scoring rules

### Strong boosters

- `+0.40` if explicit memory request
- `+0.30` if category is `decision`
- `+0.30` if category is `constraint`
- `+0.25` if category is `preference`
- `+0.20` if category is `fact`
- `+0.20` if category is `task`
- `+0.20` if it likely affects future implementation behavior
- `+0.15` if recalling it could prevent repeated mistakes or clarification
- `+0.10` if repeated or reaffirmed

### Penalties

- `-0.25` if clearly one-off and short-lived
- `-0.20` if too generic to guide future behavior
- `-0.15` if only useful in the immediate current turn
- `-0.10` if nearly duplicated by a stronger existing memory

## Derived helper flags

The extractor/scorer should produce helper flags like:

- `explicit_memory_request`
- `user_authored`
- `assistant_authored`
- `strong_category_match`
- `repeated_signal`
- `specific_statement`
- `low_specificity`
- `temporary_or_transient`
- `procedural_only`
- `affects_future_behavior`
- `prevents_future_error`
- `near_duplicate_existing`
- `conflicting_signal`

These make the scoring explainable and debuggable.

## Category-specific guidance

### preference

Boost when the candidate clearly describes how the user wants Pi to behave.

Examples:
- "Prefer concise answers"
- "Use TS + Go from the beginning"

### fact

Boost only when the fact is stable and project-relevant.

Good:
- "This project is a Pi package"

Bad:
- "We just opened THOUGHTS.md"

### decision

Treat as highly important when a choice was made and future work depends on it.

Examples:
- "Use one SQLite DB per project"
- "Use JSON-over-stdio for TS↔Go communication"

### task

Only save if the task is likely to matter beyond the current turn or next immediate message.

Good:
- "Need to define SQLite schema"

Bad:
- "Open the next file now"

### constraint

Treat constraints as high-value when they shape future work.

Examples:
- "Use as few tokens as possible"
- "Default ingestion should be algorithmic"

## Final save rule for MVP

After scoring and deduplication:

### Save if either:

1. `confidence >= 0.75` and `importance >= 0.55`

or

2. `explicit_memory_request == true` and `confidence >= 0.70`

Otherwise:
- ignore

This intentionally biases toward a cleaner memory set.

## Duplicate/update rule

If a near-duplicate existing memory is found:

- do not create a second memory item
- update the existing item instead when the new signal adds value
- append a new `memory_sources` row
- refresh `updated_at`
- optionally increase `confidence` by `+0.05`
- optionally increase `importance` by `+0.05` if the memory was reaffirmed by the user

## Contradiction rule

If a new candidate appears to contradict an existing memory:

For MVP:
- prefer not to overwrite automatically unless the new signal is explicit and clearly stronger
- if confidence is low or ambiguity exists, ignore and wait for stronger evidence

Later we can support explicit revision/conflict handling.

## Explainability requirement

Every saved memory should be explainable in terms of:

- source entries
- category
- scoring flags
- final confidence
- final importance
- why it was saved instead of ignored

This is important for trust and debugging.

## Tuning principle

The first rubric is expected to be tuned after observing real session data.
The important thing is to start with something conservative and easy to adjust.

## Session discovery and ingestion state behavior

The memory system needs a deterministic way to:

- find Pi session files relevant to the current project
- decide which parts have already been processed
- resume safely after restart or crash

### Session discovery

For MVP, session discovery should be path/project based.

Primary rule:

- only inspect Pi session files that belong to the current project/workspace

How to determine belonging:

1. resolve the current canonical project root
2. locate Pi's session storage directory
3. identify the session subdirectory associated with that project root
4. discover session files from that project-specific session location only
5. upsert discovered files into `tracked_sessions`
6. ingest from `tracked_sessions` incrementally

Why this works:

- matches Pi's project-oriented session storage
- keeps memory project-scoped
- avoids accidental cross-project ingestion

### Session directory resolution

The extension/backend should resolve session storage in this order:

1. explicit configured session directory, if known/configured
2. Pi default session storage location
3. fail with a clear status/error if session storage cannot be resolved

Important note:

- session discovery should be read-only
- Pi sessions remain the raw source of truth
- Pi Memory should never mutate Pi session files

### Which session files to ingest

For MVP:

- ingest all session files in the resolved session directory for the current project
- sort deterministically, preferably by file timestamp/name
- process each file incrementally using ingestion state

This gives us:

- current-session ingestion
- historical catch-up ingestion
- crash-safe recovery from missed turns

### Session file identity

Each session file should be tracked by its absolute canonical file path.

For MVP, `session_file` is the identity key used by:

- `tracked_sessions`
- `ingestion_state`
- `memory_sources`
- `ingestion_runs` where applicable

Later, we can extend with:

- session id extracted from content
- branch metadata
- fingerprints

### Ingestion state model

The system should remember, per session file:

- the last processed entry id
- optionally the last processed timestamp
- the last successful ingestion run
- when the file was last ingested

This is stored in `ingestion_state`.

Suggested meaning:

- one row per session file
- if no row exists, the file has never been ingested
- if a row exists, only entries after the recorded checkpoint should be considered new

### Incremental ingestion rule

When ingesting a session file:

1. load its `ingestion_state` row if present
2. read entries from the session file in order
3. skip entries up to the last known checkpoint
4. process only later entries
5. if ingestion succeeds, advance the checkpoint
6. if ingestion fails, do not advance beyond the last safely processed entry

This makes ingestion resumable and crash-tolerant.

### Checkpoint choice

For MVP, the primary checkpoint should be:

- `last_entry_id`

Optional secondary checkpoint:

- `last_entry_timestamp`

Why entry id first:

- aligns with Pi session tree entries
- more precise than timestamp alone
- avoids ambiguity when multiple entries share similar times

### What counts as safely processed

An entry should be considered safely processed only after:

- candidate extraction/scoring is complete for that entry
- any memory inserts/updates have been committed successfully
- source rows have been written successfully
- ingestion bookkeeping is ready to commit

Only then should `ingestion_state` advance.

### Transaction behavior

For MVP, use a conservative transaction strategy:

- process a session file in a DB transaction chunk, or at least in small safe batches
- write memory updates and source rows before updating `ingestion_state`
- update `ingestion_runs` accordingly

This reduces the chance of silently skipping data.

### Crash recovery behavior

If Pi crashes or Pi Memory crashes:

- Pi session files should still exist as raw source data
- on next run, Pi Memory reopens the project DB
- session discovery runs again
- files with stale or incomplete checkpoints are resumed
- entries after the last safe checkpoint are reprocessed

This is why ingestion state must advance only after successful commits.

### Session start behavior

On session start:

- resolve current project
- discover relevant session files
- run catch-up ingestion for any files with unprocessed entries
- then perform memory recall for the current session

This allows recovery from missed automatic ingestion.

### After completed assistant turns

After a completed assistant turn:

- identify the active session file
- ingest the delta for that file only
- update memory state incrementally

This keeps the system fast and low-noise during normal use.

### Manual ingestion behavior

If the user runs a manual ingest command:

- discover all relevant session files for the current project
- upsert them into `tracked_sessions`
- process all tracked files incrementally
- report how many files/entries were scanned and how many memories were created or updated

### Edge cases to handle

- session directory missing
- project has no session files yet
- session file deleted after being indexed
- checkpoint entry id no longer found in file
- migrated/compacted/branched session structures

For MVP fallback behavior:

- if checkpoint cannot be matched reliably, reprocess the whole session file conservatively and rely on deduplication

### Key design principle

Session files are append-like historical input.
The memory DB is derived state.
If in doubt, prefer safe reprocessing with deduplication over risky skipping.

## Safety net: raw session search fallback

In addition to structured memory retrieval, the system should support a fallback path that searches the project's raw Pi session history directly.

Why this matters:

- structured memory will intentionally be selective
- some useful information may never be promoted into memory
- ingestion/scoring may miss valuable context
- users should still be able to recover historical information from raw sessions

Core idea:

- structured memory is the primary recall layer
- raw session search is the fallback/safety-net layer

What this means in practice:

- if structured memory is insufficient, the tool can search the project's session files directly
- manual commands should eventually expose raw session search explicitly
- raw session search should remain project-scoped
- raw session search should be read-only and should not mutate Pi sessions

Important design distinction:

- structured memory = normalized, curated, durable, low-noise
- raw session search = historical evidence, broader recall, noisier but more complete

What we should not assume:

- Pi may contain summary-like entries in some sessions
- but we should not rely on Pi having a clean summary for every session
- if we want dependable session summaries later, we should create/store our own summary/index layer

Suggested future commands/capabilities:

- `/pi-memory-search-sessions <query>`
- memory source inspection commands
- ingestion/debug commands that explain why something was or was not stored as memory

Design principle:

If curated memory is too conservative, raw session search gives us a trustworthy fallback without bloating the main memory database.

## Retrieval and recall behavior

The recall system should be helpful, selective, and quiet.
Its job is not to dump stored memory into every session, but to surface a small number of relevant memories at the right moments.

### Core principle

Recall should optimize for:

- relevance
- low noise
- low token usage
- user trust
- easy inspectability

### Two retrieval modes

#### 1. Automatic recall

Triggered by Pi/session lifecycle moments.

Purpose:

- help Pi start a session with useful context
- remind Pi of durable project constraints, preferences, and decisions
- do so without overwhelming context

#### 2. Manual retrieval

Triggered by user commands.

Purpose:

- let the user inspect/search memory directly
- support debugging and trust
- allow deeper exploration than automatic recall

## Automatic recall behavior

### Primary trigger

For MVP, automatic recall should happen:

- on session start
- after catch-up ingestion has completed

This ensures recall works from the freshest known memory state.

### Optional later trigger

Potential future trigger:

- after explicit user memory capture
- after especially important new saved memories

But for MVP, avoid too many automatic recall points.

### What should be eligible for automatic recall

Default eligible categories:

- `constraint`
- `decision`
- `preference`
- selected high-value `fact`
- selected `task` items only if still active and likely relevant

Less likely to auto-recall:

- low-importance facts
- stale tactical tasks
- suppressed memories

### Automatic recall filters

A memory should only be considered for automatic recall if:

- `status = active`
- confidence is above a minimum threshold
- importance is above a minimum threshold
- it is not stale/irrelevant by simple heuristics
- it belongs to the current project

Suggested thresholds for MVP:

- `confidence >= 0.70`
- `importance >= 0.65`

### Automatic recall ranking factors

Rank higher when:

- category is `constraint`, `decision`, or `preference`
- importance is high
- confidence is high
- memory was reaffirmed recently
- memory likely affects future implementation behavior
- memory prevents likely mistakes or repeated clarification

Rank lower when:

- task is stale or already resolved
- fact is generic
- memory has not proven useful or has weak evidence

### Automatic recall limits

To keep token usage low, automatic recall must be capped.

Suggested MVP caps:

- target 3-5 memories
- hard maximum 7 memories
- if many candidates exist, prefer the top-ranked small set

### Automatic recall formatting

Automatic recall should be concise.

Preferred format:

- short bullet list
- one line per memory when possible
- include category label only if useful
- avoid long source excerpts in the injected context

Example:

```text
Relevant project memory:
- Decision: Use one SQLite database per project.
- Constraint: Keep token usage low; default ingestion should be algorithmic.
- Preference: Use a TypeScript Pi extension with a Go backend from the start.
```

### Source transparency

The injected automatic recall should be concise, but the system should still let the user inspect where a memory came from via manual commands.

## Manual retrieval behavior

Manual retrieval should be more flexible and more verbose than automatic recall.

Suggested MVP commands:

- `/pi-memory-list`
- `/pi-memory-search <query>`
- `/pi-memory-status`
- `/pi-memory-forget <memoryId or query>`

### `/pi-memory-list`

Purpose:

- list stored memories for the current project
- optionally filter by category/status later

Default behavior:

- show active memories ordered by importance/updated time
- use compact output
- show memory id, category, summary, and maybe confidence/importance

### `/pi-memory-search <query>`

Purpose:

- search memory summaries/details for the current project

MVP search behavior:

- simple case-insensitive text search over summary/details
- rank exact/strong matches above looser matches
- show compact results with ids

### `/pi-memory-status`

Purpose:

- show current project memory status
- help users trust/debug the system

Suggested output:

- whether project is initialized
- project db path
- number of active memories
- last ingestion time
- number of tracked session files

### `/pi-memory-forget`

Purpose:

- let the user suppress/remove memories they do not want kept

MVP behavior:

- prefer changing status rather than hard delete at first
- allow delete-like behavior later if needed

## Recall ranking model

Use a deterministic ranking score for retrieval.

Suggested retrieval score inputs:

- importance
- confidence
- category weight
- recency of reaffirmation/update
- active status
- optional query match score for manual search

### Suggested category weights for automatic recall

- `constraint`: 1.00
- `decision`: 0.95
- `preference`: 0.90
- `fact`: 0.70
- `task`: 0.60

These are ranking weights, not creation scores.

### Staleness handling

For MVP, use simple staleness rules:

- tasks may rank down over time if never reaffirmed
- suppressed memories never auto-recall
- facts/decisions/constraints remain eligible longer

### What should not happen

Avoid:

- injecting all memories blindly
- dumping source excerpts into automatic recall
- surfacing suppressed or low-confidence items
- re-showing too many repetitive items every session start if nothing changed

## Anti-noise principle

Automatic recall should be intentionally conservative.
If there is doubt about whether a memory is useful enough to inject automatically, skip it.

Manual commands can always expose more.

## User trust principle

The user should always be able to answer:

- what memory was recalled?
- why was it recalled?
- where did it come from?
- how can I remove it?

That is essential for long-term trust.

## Go binary command contract

The Go backend should expose a small, versioned command interface over JSON-over-stdio.
The contract should be stable, explicit, and easy to test independently from Pi.

### Envelope design

Every request should follow this shape:

```json
{
  "version": 1,
  "command": "init_project",
  "payload": {}
}
```

Every successful response should follow this shape:

```json
{
  "ok": true,
  "result": {}
}
```

Every failed response should follow this shape:

```json
{
  "ok": false,
  "error": {
    "code": "SOME_ERROR_CODE",
    "message": "Human-readable error message",
    "details": {}
  }
}
```

### Transport rules

- stdin: one JSON request
- stdout: one JSON response
- stderr: logs/debug info only
- exit code `0`: request handled and JSON response returned
- non-zero exit code: transport/process-level failure

### Core MVP commands

#### 1. `init_project`

Purpose:

- initialize Pi Memory for the current project
- create/update `projects.json`
- create project directory
- create `project.json`
- initialize `memory.db`

Suggested payload:

```json
{
  "projectPath": "/absolute/project/path",
  "storageBaseDir": "/Users/reld/.pi-memory",
  "projectName": "pi-memory"
}
```

Suggested result:

```json
{
  "projectId": "pi-memory-1a2b3c4d",
  "projectDir": "/Users/reld/.pi-memory/pi-memory-1a2b3c4d",
  "projectFile": "/Users/reld/.pi-memory/pi-memory-1a2b3c4d/project.json",
  "dbPath": "/Users/reld/.pi-memory/pi-memory-1a2b3c4d/memory.db",
  "created": true
}
```

Possible errors:

- `PROJECT_ALREADY_INITIALIZED`
- `INVALID_PROJECT_PATH`
- `INVALID_STORAGE_BASE_DIR`
- `INIT_FAILED`

#### 2. `get_project`

Purpose:

- resolve the current project in the registry
- return project metadata if initialized

Suggested payload:

```json
{
  "projectPath": "/absolute/project/path",
  "storageBaseDir": "/Users/reld/.pi-memory"
}
```

Suggested result:

```json
{
  "initialized": true,
  "project": {
    "projectId": "pi-memory-1a2b3c4d",
    "name": "pi-memory",
    "projectPath": "/Users/reld/Code/Playground/PI/pi-memory",
    "projectDir": "/Users/reld/.pi-memory/pi-memory-1a2b3c4d",
    "projectFile": "/Users/reld/.pi-memory/pi-memory-1a2b3c4d/project.json",
    "dbPath": "/Users/reld/.pi-memory/pi-memory-1a2b3c4d/memory.db"
  }
}
```

#### 3. `project_status`

Purpose:

- report project memory status for the current project

Suggested payload:

```json
{
  "projectPath": "/absolute/project/path",
  "storageBaseDir": "/Users/reld/.pi-memory"
}
```

Suggested result:

```json
{
  "initialized": true,
  "projectId": "pi-memory-1a2b3c4d",
  "dbPath": "/Users/reld/.pi-memory/pi-memory-1a2b3c4d/memory.db",
  "activeMemoryCount": 12,
  "trackedSessionCount": 4,
  "lastIngestedAt": "2026-04-18T12:00:00Z"
}
```

#### 4. `ingest_sessions`

Purpose:

- discover relevant sessions for the project
- upsert them into `tracked_sessions`
- process session deltas incrementally
- create/update memories and sources

Suggested payload:

```json
{
  "projectPath": "/absolute/project/path",
  "storageBaseDir": "/Users/reld/.pi-memory",
  "trigger": "auto_turn",
  "sessionDir": "/Users/reld/.pi/agent/sessions",
  "activeSessionFile": "/path/to/current/session.jsonl"
}
```

Notes:

- `activeSessionFile` is optional but useful for fast turn-based ingestion
- `trigger` values include `auto_turn`, `session_start_catchup`, `manual`, `explicit_user`

Suggested result:

```json
{
  "runId": "run_01",
  "trackedSessionsDiscovered": 4,
  "sessionFilesProcessed": 1,
  "entriesSeen": 24,
  "candidatesFound": 5,
  "memoriesCreated": 2,
  "memoriesUpdated": 1,
  "memoriesIgnored": 2,
  "lastIngestedAt": "2026-04-18T12:00:00Z"
}
```

#### 5. `list_memories`

Purpose:

- return memories for the current project

Suggested payload:

```json
{
  "projectPath": "/absolute/project/path",
  "storageBaseDir": "/Users/reld/.pi-memory",
  "status": "active",
  "limit": 50
}
```

Suggested result:

```json
{
  "items": [
    {
      "memoryId": "mem_01",
      "category": "decision",
      "summary": "Use one SQLite database per project.",
      "status": "active",
      "confidence": 0.96,
      "importance": 0.90,
      "updatedAt": "2026-04-18T12:00:00Z"
    }
  ]
}
```

#### 6. `search_memories`

Purpose:

- search structured memories for the current project

Suggested payload:

```json
{
  "projectPath": "/absolute/project/path",
  "storageBaseDir": "/Users/reld/.pi-memory",
  "query": "sqlite per project",
  "limit": 20
}
```

Suggested result:

```json
{
  "items": [
    {
      "memoryId": "mem_01",
      "category": "decision",
      "summary": "Use one SQLite database per project.",
      "score": 0.95
    }
  ]
}
```

#### 7. `forget_memory`

Purpose:

- suppress or forget a memory item

Suggested payload:

```json
{
  "projectPath": "/absolute/project/path",
  "storageBaseDir": "/Users/reld/.pi-memory",
  "memoryId": "mem_01",
  "mode": "suppress"
}
```

Suggested result:

```json
{
  "memoryId": "mem_01",
  "status": "suppressed",
  "updatedAt": "2026-04-18T12:00:00Z"
}
```

#### 8. `recall_memories`

Purpose:

- return the top ranked memories for automatic session-start recall

Suggested payload:

```json
{
  "projectPath": "/absolute/project/path",
  "storageBaseDir": "/Users/reld/.pi-memory",
  "limit": 5
}
```

Suggested result:

```json
{
  "items": [
    {
      "memoryId": "mem_01",
      "category": "decision",
      "summary": "Use one SQLite database per project.",
      "confidence": 0.96,
      "importance": 0.90,
      "recallScore": 0.94
    }
  ]
}
```

#### 9. `search_sessions`

Purpose:

- search the project's raw tracked session files directly as a fallback safety net

Suggested payload:

```json
{
  "projectPath": "/absolute/project/path",
  "storageBaseDir": "/Users/reld/.pi-memory",
  "query": "sqlite per project",
  "limit": 20
}
```

Suggested result:

```json
{
  "items": [
    {
      "sessionFile": "/path/to/session.jsonl",
      "entryId": "abc123",
      "role": "user",
      "excerpt": "We should use one SQLite DB per project.",
      "score": 0.91
    }
  ]
}
```

### Common payload pattern

Most project-scoped commands should accept:

- `projectPath`
- `storageBaseDir`

This keeps the TypeScript side simple.

### Common error codes

Suggested shared error codes:

- `PROJECT_NOT_INITIALIZED`
- `PROJECT_ALREADY_INITIALIZED`
- `PROJECT_NOT_FOUND`
- `INVALID_PROJECT_PATH`
- `INVALID_STORAGE_BASE_DIR`
- `SESSION_DIR_NOT_FOUND`
- `SESSION_DISCOVERY_FAILED`
- `INGEST_FAILED`
- `MEMORY_NOT_FOUND`
- `SEARCH_FAILED`
- `DB_ERROR`
- `INTERNAL_ERROR`

### Design principles

- commands should be project-scoped and explicit
- results should be structured for easy TS formatting
- errors should be machine-readable and user-mappable
- the contract should stay small in MVP and grow only when needed

## TypeScript command and Pi API surface

The TypeScript extension should present a small, clear Pi-facing surface.
It should hide backend complexity and make the memory system feel native inside Pi.

### Core Pi commands for MVP

#### `/pi-memory-init`

Purpose:

- explicitly initialize memory for the current project
- guide the user through setup if needed
- call Go `init_project`

Expected UX:

- resolve current project root
- show or confirm storage base dir
- initialize project memory
- report created paths and status

Typical output:

- initialized or already initialized
- project id
- db path
- project dir

#### `/pi-memory-status`

Purpose:

- show memory status for the current project
- help the user inspect/debug configuration and health

Backed by:

- Go `project_status`

Expected output:

- initialized or not
- project path
- project id
- db path
- active memory count
- tracked session count
- last ingested time

#### `/pi-memory-ingest`

Purpose:

- manually trigger ingestion for the current project
- useful for debugging, recovery, or explicit refresh

Backed by:

- Go `ingest_sessions`

Expected output:

- run id
- sessions discovered/processed
- entries seen
- candidates found
- memories created/updated/ignored

#### `/pi-memory-list`

Purpose:

- list stored memories for the current project

Backed by:

- Go `list_memories`

Expected output:

- compact list of memories with id, category, summary, and maybe score hints

#### `/pi-memory-search <query>`

Purpose:

- search structured memories for the current project

Backed by:

- Go `search_memories`

Expected output:

- ranked matching memory items

#### `/pi-memory-search-sessions <query>`

Purpose:

- search raw project session history as a fallback safety net

Backed by:

- Go `search_sessions`

Expected output:

- ranked excerpts from matching session entries
- enough context to help recovery, without dumping huge transcripts

#### `/pi-memory-forget <memoryId>`

Purpose:

- suppress/forget a memory item

Backed by:

- Go `forget_memory`

Expected UX:

- show target memory
- confirm destructive action when appropriate
- prefer status changes over hard deletes in MVP

#### `/pi-memory-remember <text>`

Purpose:

- explicit user-directed memory capture
- fast path for “please remember this” outside passive ingestion

Implementation options:

- either call `ingest_sessions` with `trigger = explicit_user` after writing a visible custom entry/message
- or add a future dedicated backend command if needed

MVP recommendation:

- keep this command in the TS layer as a convenience UX
- pass the explicit signal through the existing ingestion pipeline where possible

### Session lifecycle behavior in the extension

#### On session start

The extension should:

1. resolve current project
2. check whether project memory is initialized
3. if initialized, call Go `ingest_sessions` with `trigger = session_start_catchup`
4. call Go `recall_memories`
5. inject or display concise relevant memory

#### After completed assistant turns

The extension should:

1. determine the active session file if available
2. call Go `ingest_sessions` with `trigger = auto_turn`
3. do this quietly unless there is an error or useful status worth showing

#### On session shutdown

For MVP, shutdown behavior can remain minimal.

Possible behavior:

- no-op by default
- optionally flush a final ingestion pass later if needed

### What should be exposed as Pi tools vs commands

For MVP, prefer **commands first**.

Why:

- user-facing memory management is explicit
- easier to debug
- less risk of the model overusing memory operations
- cleaner control while the system is young

Suggested split:

#### Commands

- `/pi-memory-init`
- `/pi-memory-status`
- `/pi-memory-ingest`
- `/pi-memory-list`
- `/pi-memory-search`
- `/pi-memory-search-sessions`
- `/pi-memory-forget`
- `/pi-memory-remember`

#### Possible later LLM-callable tools

Only after the system is stable, maybe expose limited tools such as:

- `memory_search`
- `memory_remember`

But avoid this in the earliest MVP unless we have a strong reason.

### TypeScript wrapper responsibilities

The extension should include a small internal service layer that:

- resolves project path
- resolves storage base dir
- locates the Go binary
- sends JSON requests to Go
- parses responses
- maps backend errors to user-friendly Pi messages
- formats output for commands and automatic recall

### Command design principle

Every command should answer one of these clearly:

- initialize memory
- inspect memory
- refresh memory
- search memory
- search raw sessions
- remove/suppress memory
- explicitly save memory

That keeps the Pi UX understandable.

## Actual SQL DDL for MVP

The first schema should be simple, explicit, and easy to migrate.
Use SQLite-friendly types and ISO timestamp strings for readability.

### Pragmas

Suggested initialization pragmas:

```sql
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
```

### `tracked_sessions`

```sql
CREATE TABLE IF NOT EXISTS tracked_sessions (
  session_file TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  session_id TEXT,
  session_name TEXT,
  first_seen_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  last_ingested_at TEXT,
  status TEXT NOT NULL CHECK (status IN ('active', 'missing', 'stale'))
);
```

Suggested indexes:

```sql
CREATE INDEX IF NOT EXISTS idx_tracked_sessions_project_id
  ON tracked_sessions(project_id);

CREATE INDEX IF NOT EXISTS idx_tracked_sessions_status
  ON tracked_sessions(status);

CREATE INDEX IF NOT EXISTS idx_tracked_sessions_last_ingested_at
  ON tracked_sessions(last_ingested_at);
```

### `memory_items`

```sql
CREATE TABLE IF NOT EXISTS memory_items (
  memory_id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  category TEXT NOT NULL CHECK (category IN ('preference', 'fact', 'decision', 'task', 'constraint')),
  summary TEXT NOT NULL,
  details TEXT,
  status TEXT NOT NULL CHECK (status IN ('active', 'suppressed', 'forgotten')),
  source_type TEXT NOT NULL CHECK (source_type IN ('auto', 'explicit_user')),
  confidence REAL NOT NULL CHECK (confidence >= 0.0 AND confidence <= 1.0),
  importance REAL NOT NULL CHECK (importance >= 0.0 AND importance <= 1.0),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
```

Suggested indexes:

```sql
CREATE INDEX IF NOT EXISTS idx_memory_items_project_id
  ON memory_items(project_id);

CREATE INDEX IF NOT EXISTS idx_memory_items_category
  ON memory_items(category);

CREATE INDEX IF NOT EXISTS idx_memory_items_status
  ON memory_items(status);

CREATE INDEX IF NOT EXISTS idx_memory_items_source_type
  ON memory_items(source_type);

CREATE INDEX IF NOT EXISTS idx_memory_items_updated_at
  ON memory_items(updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_memory_items_importance
  ON memory_items(importance DESC);
```

### `memory_sources`

```sql
CREATE TABLE IF NOT EXISTS memory_sources (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  memory_id TEXT NOT NULL,
  session_file TEXT NOT NULL,
  entry_id TEXT,
  entry_role TEXT,
  excerpt TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (memory_id) REFERENCES memory_items(memory_id) ON DELETE CASCADE,
  FOREIGN KEY (session_file) REFERENCES tracked_sessions(session_file) ON DELETE CASCADE
);
```

Suggested indexes:

```sql
CREATE INDEX IF NOT EXISTS idx_memory_sources_memory_id
  ON memory_sources(memory_id);

CREATE INDEX IF NOT EXISTS idx_memory_sources_session_file
  ON memory_sources(session_file);

CREATE INDEX IF NOT EXISTS idx_memory_sources_entry_id
  ON memory_sources(entry_id);
```

### `ingestion_runs`

```sql
CREATE TABLE IF NOT EXISTS ingestion_runs (
  run_id TEXT PRIMARY KEY,
  started_at TEXT NOT NULL,
  finished_at TEXT,
  status TEXT NOT NULL CHECK (status IN ('running', 'completed', 'failed')),
  trigger TEXT NOT NULL CHECK (trigger IN ('auto_turn', 'session_start_catchup', 'manual', 'explicit_user')),
  session_file TEXT,
  entries_seen INTEGER NOT NULL DEFAULT 0,
  candidates_found INTEGER NOT NULL DEFAULT 0,
  memories_created INTEGER NOT NULL DEFAULT 0,
  memories_updated INTEGER NOT NULL DEFAULT 0,
  memories_ignored INTEGER NOT NULL DEFAULT 0,
  error_message TEXT,
  FOREIGN KEY (session_file) REFERENCES tracked_sessions(session_file) ON DELETE SET NULL
);
```

Suggested indexes:

```sql
CREATE INDEX IF NOT EXISTS idx_ingestion_runs_status
  ON ingestion_runs(status);

CREATE INDEX IF NOT EXISTS idx_ingestion_runs_trigger
  ON ingestion_runs(trigger);

CREATE INDEX IF NOT EXISTS idx_ingestion_runs_session_file
  ON ingestion_runs(session_file);

CREATE INDEX IF NOT EXISTS idx_ingestion_runs_started_at
  ON ingestion_runs(started_at DESC);
```

### `ingestion_state`

```sql
CREATE TABLE IF NOT EXISTS ingestion_state (
  session_file TEXT PRIMARY KEY,
  last_entry_id TEXT,
  last_entry_timestamp TEXT,
  last_ingested_at TEXT NOT NULL,
  last_run_id TEXT,
  FOREIGN KEY (session_file) REFERENCES tracked_sessions(session_file) ON DELETE CASCADE,
  FOREIGN KEY (last_run_id) REFERENCES ingestion_runs(run_id) ON DELETE SET NULL
);
```

Suggested indexes:

```sql
CREATE INDEX IF NOT EXISTS idx_ingestion_state_last_ingested_at
  ON ingestion_state(last_ingested_at DESC);
```

### Optional later: `memory_tags`

Not required for first working version, but likely useful later.

```sql
CREATE TABLE IF NOT EXISTS memory_tags (
  memory_id TEXT NOT NULL,
  tag TEXT NOT NULL,
  PRIMARY KEY (memory_id, tag),
  FOREIGN KEY (memory_id) REFERENCES memory_items(memory_id) ON DELETE CASCADE
);
```

### Migration strategy direction

For MVP:

- maintain a simple schema version table
- run ordered SQL migrations at startup/init time
- never assume a blank DB after first release

Suggested schema version table:

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL
);
```

### DDL principles

- prefer readable TEXT ids and timestamps
- keep constraints explicit
- use foreign keys for traceability integrity
- start conservative; evolve with migrations later

## Package and repository structure options

Because this project is a distributable Pi package containing both a TypeScript extension and a Go backend binary, the repository structure should make these concerns clear:

- Pi package resources
- TypeScript extension source
- Go backend source
- built/distributed artifacts
- development tooling

We also want the structure to work well with VitePlus for the TypeScript side.

## Recommended high-level direction

Use a **single repository** with a clear split between:

- package resources consumed by Pi
- TypeScript extension project
- Go backend project
- scripts/build output

This keeps development simple while still separating concerns.

## Option A — simple single-package layout

```text
pi-memory/
  AGENTS.md
  THOUGHTS.md
  TODOS.md
  package.json
  tsconfig.json
  vite.config.ts
  extensions/
    pi-memory.ts
  go/
    cmd/
      pi-memory-backend/
        main.go
    internal/
      ...
  scripts/
    build-go.*
    package.*
  vendor/
    bin/
      pi-memory-backend-<platform>
```
```

### Pros
- simple mental model
- Pi package resources are obvious
- easy to publish as one package
- minimal nesting

### Cons
- TS project files and package resources are mixed together
- build artifacts may get messy
- extension source and packaged extension may end up being the same file tree

## Option B — source + package-resources split

```text
pi-memory/
  AGENTS.md
  THOUGHTS.md
  TODOS.md
  package.json
  src/
    extension/
      index.ts
      commands/
      services/
      types/
    shared/
      ...
  extensions/
    pi-memory.ts
  go/
    cmd/
      pi-memory-backend/
        main.go
    internal/
      config/
      db/
      ingest/
      recall/
      sessions/
      search/
  scripts/
    build-go.*
    sync-extension.*
  dist/
    ...
  resources/
    bin/
      <platform binaries>
```

### Pros
- cleaner separation between source code and Pi package entrypoints
- easier TS project organization as complexity grows
- good fit for VitePlus and build tooling
- easier to test/refactor extension internals

### Cons
- requires a small packaging step to expose the final extension entry in `extensions/`
- slightly more structure to maintain

## Option C — monorepo/workspace style

```text
pi-memory/
  AGENTS.md
  THOUGHTS.md
  TODOS.md
  package.json
  packages/
    pi-extension/
      package.json
      src/
      ...
    go-bridge/
      ...
  extensions/
    pi-memory.ts
  go/
    cmd/
    internal/
  scripts/
  dist/
```

### Pros
- strongest separation
- scales well if the project becomes large
- could support separate reusable packages later

### Cons
- probably too heavy for MVP
- more tooling complexity than we need right now
- may slow down iteration early

## Recommendation

I recommend **Option B**.

Decision taken:

- We will use **Option B: source + package-resources split**.
- We will use a **thin stable entrypoint** in `extensions/pi-memory.ts`.
- The real TypeScript implementation will live in `src/extension/`.
- We will use **VitePlus** for TypeScript project setup/workflow.

Important nuance:

- VitePlus is a tooling choice for this project.
- It is **not** a hard requirement of Pi packages themselves.
- Pi packages can work without VitePlus, but VitePlus is a strong fit for this project's complexity and chosen structure.

Why:

- it gives us a clean TypeScript codebase layout
- it fits VitePlus better than a flat mixed tree
- it still keeps the Pi package simple
- it avoids premature monorepo complexity
- it leaves room for growth without becoming messy fast

## Recommended concrete structure (Option B refined)

```text
pi-memory/
  AGENTS.md
  THOUGHTS.md
  TODOS.md

  package.json
  tsconfig.json
  vite.config.ts
  .gitignore
  README.md

  src/
    extension/
      index.ts
      commands/
        init.ts
        status.ts
        ingest.ts
        list.ts
        search.ts
        search-sessions.ts
        forget.ts
        remember.ts
      services/
        backend.ts
        project.ts
        storage.ts
        recall.ts
      types/
        backend.ts
        memory.ts
        project.ts
      util/
        formatting.ts
        errors.ts

  extensions/
    pi-memory.ts

  go/
    go.mod
    go.sum
    cmd/
      pi-memory-backend/
        main.go
    internal/
      api/
      config/
      db/
      migrations/
      projects/
      sessions/
      ingest/
      memories/
      search/
      recall/
      util/

  scripts/
    build-go.sh
    build-go.ts
    package.ts
    dev.ts

  resources/
    bin/
      darwin-arm64/
      darwin-x64/
      linux-arm64/
      linux-x64/

  dist/
    extension/
    package/
```

## Directory roles

### `src/extension/`
- TypeScript source for the Pi extension
- all commands and Pi integration logic live here

### `extensions/`
- Pi-visible extension entrypoint(s)
- likely thin wrapper(s) that load compiled/bundled code from `dist/` or directly import source depending on dev strategy

### `go/`
- Go backend source
- DB, ingestion, search, recall, and command contract implementation

### `resources/bin/`
- packaged backend binaries by platform
- used by the TS extension at runtime

### `scripts/`
- local build/packaging automation

### `dist/`
- generated output only
- not hand-edited

## Entry point decision

Decision taken:

- `extensions/pi-memory.ts` will be a thin stable entrypoint
- the real implementation will live in `src/extension/`
- this keeps Pi package discovery clean while preserving a proper application structure

That keeps the Pi package shape clean while preserving a proper TS app structure.

## Packaging principle

The final distributable Pi package should clearly contain:

- `extensions/` for Pi discovery
- runtime JS/TS needed by the extension
- bundled or included Go binaries/resources
- package metadata in `package.json`

The repo structure should optimize for development first, without making distribution awkward.

## Go binary inclusion, distribution, and platform support

Because this Pi package includes a Go backend, we need a distribution strategy that is practical for users and realistic for us to maintain.

## Recommendation

For MVP, the best default strategy is:

- ship **prebuilt binaries** inside the package
- resolve the correct binary at runtime based on OS/architecture
- do **not** require users to have Go installed
- treat build-on-install as a possible future/dev fallback, not the primary installation path

This gives the best user experience for a Pi package.

## Why prebuilt binaries are the right default

Pros:

- easiest installation experience for users
- no Go toolchain requirement on user systems
- predictable runtime behavior
- good fit for a packaged Pi extension
- avoids install-time compilation issues

Cons:

- package size increases
- release process becomes more involved
- we must build and test binaries per supported platform

Even with those downsides, this is still the better MVP tradeoff for a distributed package.

## Why build-on-install should not be the default

Build-on-install would mean users may need:

- Go installed
- a compatible Go version
- working local build environment
- enough permissions/tools for compilation

That creates avoidable friction.

For a Pi package, the smoother path is to include binaries rather than ask users to compile them.

## Runtime binary resolution

The TypeScript extension should resolve the backend binary by:

1. detecting platform
   - OS
   - CPU architecture
2. mapping to a packaged binary path
3. verifying the binary exists and is executable
4. failing with a clear error if the current platform is unsupported

Suggested packaged layout:

```text
resources/
  bin/
    darwin-arm64/
      pi-memory-backend
    darwin-x64/
      pi-memory-backend
    linux-arm64/
      pi-memory-backend
    linux-x64/
      pi-memory-backend
```

Later, Windows can use:

```text
resources/
  bin/
    windows-x64/
      pi-memory-backend.exe
```

## Supported platforms for MVP

Recommended MVP support:

- `darwin-arm64`
- `darwin-x64`
- `linux-arm64`
- `linux-x64`

Possible later support:

- `windows-x64`
- `windows-arm64`

### Why this MVP platform set

- good coverage for macOS and Linux developers
- matches likely early adopter environments
- avoids Windows complexity in the first release if we want to move faster

If you want broadest reach from day one, Windows can be included later once the core is stable.

## Installation flow for users

Ideal user experience:

1. install the Pi package
2. Pi loads the extension
3. extension resolves packaged backend binary automatically
4. user runs `/pi-memory-init`
5. no separate Go installation required

This is the simplest and most trustworthy flow.

## Local development workflow

For development, we can support either:

- packaged prebuilt binaries in `resources/bin/...`
- or a dev override path/environment variable pointing to a locally built Go binary

This gives us fast iteration without forcing production behavior during local dev.

Example optional dev override idea:

- env var like `PI_MEMORY_BACKEND_PATH`
- if set, TS uses that binary instead of packaged resources

This is optional but useful.

## Update strategy

When the package updates:

- the new package version ships updated binaries
- the extension resolves the new packaged backend automatically
- per-project DB migrations run as needed when the backend opens a DB

This keeps runtime/backend versions aligned with the package version.

## Packaging principle

The package should be self-contained enough that a normal user can install and run it without separately installing Go.

That is the clearest package UX.

## Final recommendation summary

For MVP:

- include **prebuilt binaries** in the package
- support **macOS + Linux** first
- add **Windows later** if desired
- allow optional dev override for local development
- do **not** make build-on-install the default path

## Configuration model

The config model should stay explicit, minimal, and easy to reason about.
For MVP, configuration should mainly support:

- memory storage location
- backend binary resolution overrides for development
- session storage resolution overrides when needed
- a small number of behavior toggles

## Configuration scope

We should support two conceptual scopes:

### 1. Global config

Applies across projects and installations.
Useful for:

- base storage directory
- backend override path
- default behavior toggles

### 2. Project-specific runtime state

Stored in `project.json`, not a separate free-form config file for MVP.
Useful for:

- project path
- project id
- DB path
- metadata/state tied to initialization

For MVP, avoid introducing too many config files.

## Recommended config sources

Priority order should be:

1. explicit command arguments / interactive setup choices
2. environment variables (mainly for development/override)
3. package/global config file if we add one
4. defaults

This keeps behavior controllable without being confusing.

## Recommended MVP config fields

### `storageBaseDir`

Purpose:

- base directory where all project memory folders live

Default:

- `~/.pi-memory`

Notes:

- user chooses/accepts this during `/pi-memory-init`
- saved as part of initialized project metadata/registry behavior

### `backendPathOverride`

Purpose:

- force use of a specific backend binary path
- mainly for development/testing

Default:

- unset

Suggested env var:

- `PI_MEMORY_BACKEND_PATH`

Behavior:

- if set, TS uses this instead of packaged binaries

### `sessionDirOverride`

Purpose:

- override Pi session storage resolution if needed

Default:

- unset

Suggested env var:

- `PI_MEMORY_SESSION_DIR`

Behavior:

- if set, backend/extension uses this session dir instead of auto-resolving Pi defaults

### `autoIngest`

Purpose:

- whether automatic ingestion after completed assistant turns is enabled

Default:

- `true`

### `autoRecall`

Purpose:

- whether automatic recall on session start is enabled

Default:

- `true`

### `recallLimit`

Purpose:

- cap how many memories are auto-recalled

Default:

- `5`

### `rawSessionSearchEnabled`

Purpose:

- whether raw session fallback search commands are enabled

Default:

- `true`

## Possible config file shape

If we add a package/global config file later, it could look like:

```json
{
  "storageBaseDir": "~/.pi-memory",
  "autoIngest": true,
  "autoRecall": true,
  "recallLimit": 5,
  "rawSessionSearchEnabled": true
}
```

For MVP, we do not necessarily need a separate config file immediately if the extension can work from defaults + init choices + env overrides.

## Recommended MVP posture

For MVP:

- keep config minimal
- rely on sane defaults
- allow explicit setup choices during `/pi-memory-init`
- support env overrides for development and advanced cases
- avoid building a big config system too early

## TS-side config resolution responsibility

The TypeScript extension should resolve effective config by:

1. reading explicit command/context values
2. checking env overrides
3. applying defaults
4. passing resolved values to the Go backend

This keeps the backend contract simple and explicit.

## Design principle

Configuration should help users when needed, but the package should work well with almost no manual configuration.

That is especially important for a Pi package intended to feel lightweight and local-first.

## Implementation roadmap

Now that the architecture and behavior are defined, implementation should proceed in small vertical slices.
The goal is to get a working end-to-end path early, then expand safely.

## Phase A — Scaffold the repository

Goals:

- create the Option B repository structure
- set up the TypeScript project with VitePlus
- create the Go module and command entrypoint
- add package metadata for Pi package distribution

Deliverables:

- `package.json`
- `tsconfig.json`
- `vite.config.ts`
- `src/extension/...`
- `extensions/pi-memory.ts`
- `go/go.mod`
- `go/cmd/pi-memory-backend/main.go`
- basic scripts and folders

Success condition:

- repo has the agreed structure and can build basic TS + Go artifacts

## Phase B — Backend foundation

Goals:

- implement config/path resolution helpers
- implement DB open/init logic
- implement schema migrations
- implement project registry read/write

Deliverables:

- SQLite initialization
- `schema_migrations`
- table creation for MVP schema
- `init_project`
- `get_project`
- `project_status`

Success condition:

- backend can initialize and reopen project memory correctly

## Phase C — TS ↔ Go bridge

Goals:

- implement backend binary resolution
- implement JSON-over-stdio wrapper
- implement error mapping
- wire first Pi commands

Deliverables:

- TS backend service wrapper
- `/pi-memory-init`
- `/pi-memory-status`

Success condition:

- Pi command can initialize a project and report status through the real backend

## Phase D — Session tracking and ingestion foundation

Goals:

- implement session discovery
- implement `tracked_sessions`
- implement `ingestion_runs` and `ingestion_state`
- implement incremental ingestion traversal

Deliverables:

- backend session discovery logic
- `ingest_sessions` command skeleton
- tracked session upsert
- checkpointing logic

Success condition:

- backend can discover project sessions and track ingestion progress safely

## Phase E — Memory extraction and persistence

Goals:

- implement candidate extraction
- implement deterministic scoring
- implement deduplication/update behavior
- write `memory_items` and `memory_sources`

Deliverables:

- extraction heuristics
- scoring engine
- dedupe/update logic
- working `ingest_sessions`

Success condition:

- ingesting real session deltas produces useful stored memories

## Phase F — Recall and memory inspection

Goals:

- implement structured memory retrieval
- implement automatic recall ranking
- implement user inspection commands

Deliverables:

- `list_memories`
- `search_memories`
- `recall_memories`
- `/pi-memory-list`
- `/pi-memory-search`
- session-start recall in TS

Success condition:

- users can inspect memories and Pi can recall concise relevant memory on session start

## Phase G — Raw session fallback search

Goals:

- implement direct session search across tracked project sessions
- expose fallback search command in Pi

Deliverables:

- `search_sessions`
- `/pi-memory-search-sessions`

Success condition:

- users can recover context from raw sessions when structured memory is insufficient

## Phase H — Memory management UX

Goals:

- implement forget/suppress flows
- implement explicit remember flow
- improve command ergonomics and trust/debug UX

Deliverables:

- `forget_memory`
- `/pi-memory-forget`
- `/pi-memory-remember`
- improved status/reporting output

Success condition:

- users can explicitly add and remove memory with confidence

## Phase I — Packaging and release readiness

Goals:

- bundle/include prebuilt Go binaries
- verify runtime platform resolution
- document installation and usage
- validate package contents

Deliverables:

- packaged binaries under `resources/bin/...`
- runtime binary resolver
- README
- package metadata for Pi package distribution

Success condition:

- package can be installed and used on supported platforms without Go preinstalled

## Recommended immediate build order

Start with the smallest useful end-to-end slice:

1. scaffold repo
2. implement backend DB/project init/status
3. implement TS wrapper + `/pi-memory-init` + `/pi-memory-status`
4. implement session discovery/checkpointing
5. implement ingestion heuristics
6. implement list/search/recall
7. implement raw session search fallback
8. implement remember/forget flows
9. finish packaging/distribution

## First concrete coding tasks

1. create repository structure and base files
2. create `package.json` with Pi package metadata
3. create VitePlus TS project config
4. create thin `extensions/pi-memory.ts`
5. create Go module and `main.go`
6. implement JSON request dispatcher in Go
7. implement DB init + migrations
8. implement TS backend service wrapper
9. wire `/pi-memory-init`
10. wire `/pi-memory-status`

## Initial product shape

The extension should likely do the following:

- discover or receive the current project identity
- resolve the project's configured memory database path
- read relevant Pi session files
- extract candidate memories from session history
- write approved/derived memories into that project's SQLite database
- retrieve relevant memories in future sessions
- expose commands/tools for inspecting, editing, and deleting memories

Likely memory categories:

- facts about the project
- user preferences
- recurring goals
- decisions made earlier
- unfinished tasks / follow-ups
- constraints / conventions

## Pi TUI extension/UI direction (later feature area)

We confirmed that Pi extensions can render custom TUI components and overlays.
The docs and examples show that we can build:

- popups / overlays via `ctx.ui.custom(..., { overlay: true })`
- persistent widgets above or below the editor via `ctx.ui.setWidget()`
- status indicators via `ctx.ui.setStatus()`
- custom footer/header UI
- custom interactive components with keyboard handling
- list selection UIs via built-in components like `SelectList`
- settings/toggle UIs via `SettingsList`

Relevant Pi docs/examples:
- `docs/tui.md`
- `examples/extensions/todo.ts`
- `examples/extensions/widget-placement.ts`
- `examples/extensions/overlay-test.ts`
- `examples/extensions/tools.ts`
- `examples/extensions/preset.ts`

Potential Pi Memory UI ideas for later:
- memory list popup / overlay
- searchable memory picker
- raw session search result viewer
- memory detail inspector with source excerpts
- small persistent status/widget showing memory state for the current project
- optional settings panel for memory behavior

Design principle for later:
- keep the first UI additions lightweight and explicitly user-invoked
- prefer overlays/widgets over a large always-on interface
- keep memory inspection fast, inspectable, and low-noise

## Later feature ideas to revisit

Two larger ideas worth revisiting later:
- a todo system inside `pi-memory`
- a software design/spec concept inside `pi-memory`, inspired by how `THOUGHTS.md` is being used now

These are intentionally deferred for now; core memory/retrieval quality still comes first.

## Open questions for later

- Where exactly are this project's sessions being stored right now?
- Should memory be generated directly from raw session files, or from a curated memory store?
- What should count as a memory? Facts, preferences, goals, tasks, decisions?
- How should users review, edit, or delete memories?
- When should the extension inject memories into context?
- How do we prevent noisy or overly invasive recall?
- Should memory be per-project, global, or both?
- What exact project identity/hash scheme should we use for directory naming?
- What exact fields belong in `project.json`?
- What minimal fields belong in `projects.json`?
- How should relinking work when a project directory moves or gets renamed?
- How do we distribute or install the Go binary across platforms?
