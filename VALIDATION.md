# VALIDATION

## 2026-04-18 manual validation pass

This file records manual validation results for the current `pi-memory` implementation.

## Scope of this pass

Validated:
- backend build
- backend command behavior on the real project DB
- retrieval relevance at the backend-result level
- selected backend/extension failure modes
- current gaps discovered during validation

Not fully validated in this pass:
- live Pi model tool-choice behavior inside a real chat session
- full end-to-end command UX inside Pi
- rename/move edge cases
- target-platform packaging/runtime validation

## Environment

Project:
- `/Users/reld/Code/Playground/PI/pi-memory`

Storage base dir:
- `/Users/reld/.pi-memory`

Backend binary:
- `dist/package/bin/pi-memory-backend`

Observed project status:
- initialized: `true`
- project id: `pi-memory-268fccf9`
- active memories: `11`
- tracked sessions: `3`
- last ingested at: `2026-04-18T19:44:56Z`

## Build validation

Command:

```bash
vp run build:go
```

Result:
- passed
- backend built successfully to `dist/package/bin/pi-memory-backend`

## Retrieval relevance checks

### 1. Broad recall

Command:

```json
{"version":1,"command":"recall_memories","payload":{"projectPath":"/Users/reld/Code/Playground/PI/pi-memory","storageBaseDir":"/Users/reld/.pi-memory","limit":5}}
```

Observed result summary:
- returned 5 memories
- top result was a recently created memory summarizing open TODOs
- returned the commit message preference memory
- also returned lower-value/noisier items like `keep working no` and `I love arepas`

Assessment:
- **partial pass**
- recall is functional and includes some important project context
- recall is still too noisy for a strong "where were we?" experience
- unrelated/personal memory can still rank too high for project-state recall

### 2. Remembered preference / commit message constraint

Query:
- `commit message`

Result:
- returned the expected commit message formatting preference/constraint memory

Assessment:
- **pass**
- targeted structured search works for this remembered preference

### 3. Tooling convention

Query:
- `VitePlus`

Result:
- returned the expected VitePlus memory
- also returned the architecture/package memory as a secondary related result

Assessment:
- **pass**
- this is a good example of structured memory search working as intended

### 4. Prior architecture decision

Query:
- `architecture package TypeScript Go backend`

Result:
- returned no structured memory results

Follow-up query:
- `TypeScript extension Go backend`

Result:
- returned no structured memory results

Assessment:
- **fail / weak spot**
- the architecture memory exists, but the current search path did not retrieve it for intuitive architecture-style queries
- this suggests a relevance/tokenization/query-matching weakness in structured search

### 5. Project constraints

Query:
- `constraints low token ingestion Go heuristics`

Result:
- returned no structured memory results

Assessment:
- **fail / weak spot**
- high-value constraint retrieval is currently weaker than desired
- likely needs search/ranking/query matching improvements and possibly memory cleanup

### 6. Raw session fallback search

Query:
- `privacy and control principles`

Result:
- returned relevant raw session excerpts from tracked session files
- results included assistant/tool-result excerpts referencing the still-open TODO

Assessment:
- **pass**
- raw session search works as an effective fallback layer
- it can surface relevant historical evidence when structured retrieval is insufficient

## Failure-mode checks

### 1. Uninitialized project

Command used against a fresh temp directory:
- `search_memories` with a valid query

Result:

```json
{"ok":false,"error":{"code":"PROJECT_NOT_INITIALIZED","message":"Project is not initialized"}}
```

Assessment:
- **pass**
- backend clearly reports uninitialized-project state
- extension error mapping now points users to `/pi-memory-init`

### 2. Invalid query

Command:
- `search_memories` with empty query

Result:

```json
{"ok":false,"error":{"code":"INVALID_QUERY","message":"query is required"}}
```

Assessment:
- **pass**
- backend failure is clear
- extension error mapping should convert this into a more helpful user-facing message

### 3. Missing memory id / nonexistent memory

Command:
- `forget_memory` with `memoryId = mem-does-not-exist`

Result:

```json
{"ok":false,"error":{"code":"MEMORY_NOT_FOUND","message":"memory not found"}}
```

Assessment:
- **pass**
- backend failure is clear
- extension error mapping now suggests `/pi-memory-list` or `/pi-memory-search`

### 4. Nonexistent project path for status

Command:
- `project_status` with project path `/tmp/pi-memory-uninitialized` that does not exist

Result:

```json
{"ok":false,"error":{"code":"PROJECT_STATUS_FAILED","message":"stat /tmp/pi-memory-uninitialized: no such file or directory"}}
```

Assessment:
- **mixed**
- technically correct
- user-facing UX may still need a friendlier message if status is requested from a missing path

### 5. Backend-not-found guidance

Status:
- not executed end-to-end in Pi during this pass
- reviewed statically in extension error mapping

Current mapping behavior:
- `BACKEND_NOT_FOUND` suggests `vp run build` or `PI_MEMORY_BACKEND_PATH`

Assessment:
- **partial pass via static review**
- should still be exercised end-to-end in Pi later

## Key findings

### What is working well
- backend builds cleanly
- project status works on the real DB
- targeted structured search works for some important memories
- raw session fallback search is useful
- core failure codes are clear and now map to better extension messages

### Current weak spots
1. broad recall is still noisy
2. structured retrieval misses intuitive architecture/constraint queries
3. some low-value memories still rank too highly for general recall
4. missing-path status errors could use more user-friendly handling if desired

## Retrieval improvement follow-up applied after this pass

After the initial findings, retrieval logic was adjusted in `go/internal/memories/query.go` to:
- move structured search ranking into Go-side token-aware scoring
- improve matching for architecture-style queries
- improve matching for constraint/tooling-style queries
- demote obviously noisy memories such as `I love arepas` and `keep working no`
- reduce broad-recall noise through recall-specific scoring adjustments

### Spot-check results after the improvement

Improved queries:
- `architecture package TypeScript Go backend`
  - now returns the architecture memory as the top result
- `TypeScript extension Go backend`
  - now returns the architecture memory as the top result
- `constraints low token ingestion Go heuristics`
  - now returns the low-token ingestion memory as the top result
- `how should we handle TypeScript workflows in this repo again`
  - returns the VitePlus memory as the top result
- `do you remember what I said about commit messages`
  - returns the commit-message memory as the top result

Broad recall after the improvement:
- no longer surfaces `I love arepas` or `keep working no` in the top recall set
- still shows a TODO-summary memory and some feature-idea/task memories, so recall quality is improved but not yet fully polished

## Recommended next follow-up work

1. continue improving broad recall ranking
   - further reduce TODO/feature-idea noise when the user asks broad status questions
2. run the same validation pass inside a real Pi session to verify actual tool choice
3. continue rename/move and packaging/runtime validation

## Suggested todo interpretation

Current todo state still looks appropriate:
- `Test retrieval relevance` should remain **in progress**, but it is materially improved
- `Test extension/binary failure modes` can reasonably remain **in progress** until Pi-side end-to-end validation is also done
