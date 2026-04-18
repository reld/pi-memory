# TODOS

## Status legend

- [ ] not started
- [~] in progress
- [x] done

## Phase 0 — Product definition

- [x] Define the extension's purpose in one clear paragraph
- [x] Define the MVP scope
- [x] Define what counts as a memory item
- [x] Define memory categories
- [ ] Define project scoping rules
- [ ] Define privacy and control principles
- [x] Define success criteria for the MVP

## Phase 1 — Architecture

- [x] Define the high-level architecture: TypeScript extension + Go binary
- [x] Define responsibilities of the TypeScript side
- [x] Define responsibilities of the Go side
- [x] Define the communication boundary between TS and Go
- [x] Decide whether communication is CLI-based, JSON-over-stdio, or local RPC
- [ ] Define error handling between extension and binary
- [ ] Define logging and debug strategy
- [x] Define configuration model

## Phase 2 — Project identity and storage layout

- [x] Define how a project is identified
- [x] Define how project slugs are generated
- [x] Define how project hashes are generated
- [x] Define deterministic project directory naming
- [x] Define base storage directory behavior
- [x] Define default storage path
- [x] Define user-selected storage path flow
- [x] Define global `projects.json` registry shape
- [x] Define per-project `project.json` shape
- [x] Define final on-disk layout, e.g. `~/.pi-memory/[project-slug-hash]/memory.db`
- [x] Define behavior when a DB does not exist yet
- [x] Define relinking behavior when project paths move or get renamed

## Phase 3 — SQLite schema

- [x] Define the schema for the per-project database
- [x] Define `tracked_sessions`
- [x] Define `memory_items`
- [x] Define `memory_sources`
- [x] Define `ingestion_runs`
- [x] Define tags/categories storage
- [x] Define review/status fields (active, suppressed, deleted, etc.)
- [x] Define timestamps and audit fields
- [x] Define indexes
- [x] Define migration strategy

## Phase 4 — Session ingestion design

- [x] Define how Pi session files are discovered
- [x] Define which sessions belong to a project
- [x] Define explicit project-scoped session tracking (`tracked_sessions`)
- [x] Define how already-ingested content is tracked
- [x] Define incremental ingestion strategy
- [x] Define automatic ingestion trigger after completed assistant turns
- [x] Define catch-up ingestion behavior on session start
- [x] Define explicit user-triggered memory capture behavior
- [x] Define extraction rules for candidate memories
- [x] Define deterministic scoring/filtering rules for memory candidates
- [x] Define deduplication strategy
- [x] Define confidence/scoring model
- [x] Decide when, if ever, model-assisted extraction is used beyond the algorithmic path
- [x] Define source traceability back to sessions and entry IDs
- [x] Define crash recovery / resumed ingestion behavior

## Phase 5 — Retrieval and recall

- [x] Define recall triggers
- [x] Define manual search behavior
- [x] Define automatic recall behavior
- [x] Define relevance ranking strategy
- [x] Define how many memories can be surfaced at once
- [x] Define how recalled memory is formatted for Pi
- [x] Define safeguards against noisy or invasive recall
- [ ] Define raw session search fallback behavior
- [ ] Define project-scoped raw session search command(s)
- [ ] Decide whether to store our own session summaries/index later

## Phase 6 — Pi extension UX

- [x] Define setup flow inside Pi
- [x] Define `/pi-memory-init` behavior
- [ ] Define configuration commands
- [x] Define memory inspection commands
- [x] Define memory edit/delete/forget commands
- [x] Define ingestion commands
- [~] Define custom tools exposed to the LLM
- [x] Define what happens on session start
- [x] Define what happens after completed assistant turns
- [x] Define what happens on session shutdown
- [x] Define how users review stored memory

## Phase 7 — Go backend

- [ ] Create Go module
- [x] Define binary interface contract
- [ ] Implement config loading
- [x] Implement DB initialization
- [x] Implement migrations
- [ ] Implement ingestion bookkeeping
- [ ] Implement algorithmic candidate extraction from session-derived input
- [ ] Implement deterministic candidate scoring/filtering
- [ ] Implement write operations for memory items
- [ ] Implement search/query operations
- [ ] Implement structured JSON I/O for TS integration
- [ ] Implement logging/debug output

## Phase 8 — TypeScript extension

- [x] Create extension scaffold
- [ ] Add config loading/resolution
- [ ] Add setup command
- [ ] Add project DB resolution
- [ ] Add Go binary invocation wrapper
- [ ] Add ingestion command(s)
- [ ] Add retrieval/search command(s)
- [ ] Add memory management command(s)
- [ ] Add session lifecycle hooks
- [ ] Add user-facing notifications and errors

## Phase 9 — Packaging and distribution

- [x] Define package structure
- [x] Define how the Go binary is included
- [x] Decide prebuilt binaries vs build-on-install
- [x] Define supported platforms
- [x] Define installation flow for users
- [x] Define update strategy
- [x] Define local development workflow
- [x] Confirm Option B + thin entrypoint approach
- [x] Confirm VitePlus as chosen TS tooling

## Phase 10 — Testing and validation

- [ ] Test project setup flow
- [ ] Test DB creation in custom location
- [ ] Test ingestion on real Pi sessions
- [ ] Test duplicate prevention
- [ ] Test retrieval relevance
- [ ] Test project isolation
- [ ] Test rename/move edge cases
- [ ] Test extension/binary failure modes
- [ ] Test packaging on target platforms

## Immediate next tasks

- [x] Write a one-paragraph product definition
- [x] Define the MVP feature set
- [x] Define TS vs Go responsibilities precisely
- [x] Define the TS↔Go communication model
- [x] Define project identity and DB path rules
- [x] Define `projects.json` vs `project.json` responsibilities

## Phase 11 — Implementation roadmap / execution

- [x] Define implementation roadmap by phase
- [x] Scaffold repository structure
- [x] Create `package.json` with Pi package metadata
- [x] Create VitePlus TS base config
- [x] Create thin `extensions/pi-memory.ts`
- [x] Create Go module and backend entrypoint
- [x] Implement Go JSON command dispatcher
- [x] Implement DB init + migrations
- [ ] Implement TS backend wrapper
- [ ] Wire `/pi-memory-init`
- [ ] Wire `/pi-memory-status`
