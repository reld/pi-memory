# TESTING

## Retrieval relevance test plan

Goal: verify that `pi-memory` surfaces the right historical context with low noise and that the model chooses the right memory tool for the prompt.

### Preconditions

- Pi Memory is initialized for the project
- the project already has real stored memories
- auto-ingest is enabled for normal validation unless a test explicitly disables it
- raw session search is enabled for fallback validation

### Core prompts to test

#### 1. Broad recall
Prompt:
- `where were we?`

Expected behavior:
- model should prefer `pi_memory_recall`
- response should summarize relevant prior work, not dump all memories
- should avoid raw session search unless structured recall is insufficient

#### 2. Remembered preference / constraint
Prompt:
- `do you remember what I said about commit messages?`

Expected behavior:
- model should use `pi_memory_search` or `pi_memory_recall`
- should return the commit message formatting preference/constraint
- answer should be specific and concise

#### 3. Prior architecture decision
Prompt:
- `what did we decide about the architecture for this package?`

Expected behavior:
- model should use `pi_memory_search` or `pi_memory_recall`
- should mention Pi package + TypeScript extension + Go backend
- should mention only relevant decisions

#### 4. Project conventions / constraints
Prompt:
- `what project constraints should you keep in mind?`

Expected behavior:
- model should prefer `pi_memory_recall`
- should surface high-value constraints like low-token ingestion / Go-heavy heuristics
- should avoid irrelevant facts if better constraints are available

#### 5. Tooling convention
Prompt:
- `how should we handle TypeScript workflows in this repo again?`

Expected behavior:
- model should use `pi_memory_search`
- should mention VitePlus / `vp`
- should not drift into generic npm guidance

#### 6. Fallback to raw sessions
Prompt:
- ask about a prior discussion detail that is likely present in session history but not stored as structured memory

Expected behavior:
- model should first try structured memory when appropriate
- if insufficient, it may use `pi_memory_search_sessions`
- answer should make use of session evidence without over-quoting raw history

### Negative tests

#### 7. No unnecessary memory lookup
Prompt:
- a simple local coding request with no historical dependency

Expected behavior:
- model should not call memory tools unless context clearly requires it

#### 8. No raw-session overuse
Prompt:
- a normal memory question that structured memory can answer

Expected behavior:
- model should not jump to `pi_memory_search_sessions` first

### Failure-mode tests

#### 9. Uninitialized project
Prompt:
- `where were we?`

Expected behavior:
- tool call fails clearly
- user-visible outcome should indicate `/pi-memory-init` is needed

#### 10. Raw session search disabled
Prompt:
- ask for a prior conversation detail requiring fallback

Expected behavior:
- raw search tool should fail clearly
- model should not hallucinate raw-history results

### What to record per test

For each prompt, note:
- prompt text
- which tool was used, if any
- whether the chosen tool was appropriate
- whether the returned memory was relevant
- whether important memory was missed
- whether noisy/irrelevant memory was included
- whether the final answer was actually helpful

### Pass criteria

A test pass means:
- the model chose the right tool or correctly avoided tools
- the retrieved context was relevant and concise
- the final answer used the retrieved memory correctly
- token usage stayed proportionate to the question
