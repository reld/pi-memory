package ingest

import (
	"encoding/json"
	"strings"

	"pi-memory/internal/memories"
)

type SessionEntry struct {
	ID        string          `json:"id"`
	Timestamp string          `json:"timestamp"`
	Message   *MessagePayload `json:"message,omitempty"`
}

type MessagePayload struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type textBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func ExtractCandidates(sessionFile string, entry SessionEntry) []memories.Candidate {
	if entry.Message == nil {
		return nil
	}
	role := entry.Message.Role
	if role != "user" {
		return nil
	}

	text := ExtractEntryText(entry)
	if text == "" || shouldIgnoreForMemory(text) {
		return nil
	}

	candidates := make([]memories.Candidate, 0, 5)
	if candidate, ok := explicitMemoryCandidate(sessionFile, entry, text); ok {
		candidates = append(candidates, candidate)
	}
	if candidate, ok := preferenceCandidate(sessionFile, entry, text); ok {
		candidates = append(candidates, candidate)
	}
	if candidate, ok := decisionCandidate(sessionFile, entry, text); ok {
		candidates = append(candidates, candidate)
	}
	if candidate, ok := constraintCandidate(sessionFile, entry, text); ok {
		candidates = append(candidates, candidate)
	}
	if candidate, ok := taskCandidate(sessionFile, entry, text); ok {
		candidates = append(candidates, candidate)
	}
	if candidate, ok := factCandidate(sessionFile, entry, text); ok {
		candidates = append(candidates, candidate)
	}
	return dedupeCandidates(candidates)
}

func explicitMemoryCandidate(sessionFile string, entry SessionEntry, text string) (memories.Candidate, bool) {
	phrases := []string{"remember this", "please remember", "note this", "remember that"}
	for _, phrase := range phrases {
		if containsFold(text, phrase) {
			summary := cleanupAfterPhrase(text, phrase)
			summary = bestSnippet(summary)
			if !isUsefulSummary(summary) {
				summary = bestSnippet(text)
			}
			if !isUsefulSummary(summary) {
				return memories.Candidate{}, false
			}
			return memories.Candidate{
				Category:    classifyExplicit(summary),
				Summary:     sentence(summary),
				Details:     text,
				SourceType:  "explicit_user",
				Confidence:  0.95,
				Importance:  0.90,
				EntryID:     entry.ID,
				EntryRole:   entry.Message.Role,
				Excerpt:     sentence(text),
				SessionFile: sessionFile,
			}, true
		}
	}
	return memories.Candidate{}, false
}

func preferenceCandidate(sessionFile string, entry SessionEntry, text string) (memories.Candidate, bool) {
	patterns := []string{"i prefer", "please use", "don't use", "do not use", "prefer "}
	for _, pattern := range patterns {
		if containsFold(text, pattern) {
			summary := extractSummaryFromPattern(text, pattern)
			if !isUsefulSummary(summary) {
				return memories.Candidate{}, false
			}
			return candidateFromSummary(sessionFile, entry, "preference", summary, text, 0.80, 0.80), true
		}
	}
	return memories.Candidate{}, false
}

func decisionCandidate(sessionFile string, entry SessionEntry, text string) (memories.Candidate, bool) {
	patterns := []string{"we decided", "let's go with", "we will use", "we should use", "let us use"}
	for _, pattern := range patterns {
		if containsFold(text, pattern) {
			summary := extractSummaryFromPattern(text, pattern)
			if !isUsefulSummary(summary) {
				return memories.Candidate{}, false
			}
			return candidateFromSummary(sessionFile, entry, "decision", summary, text, 0.84, 0.88), true
		}
	}
	return memories.Candidate{}, false
}

func constraintCandidate(sessionFile string, entry SessionEntry, text string) (memories.Candidate, bool) {
	patterns := []string{"must ", "never ", "always ", "keep ", "should use as few tokens as possible", "default ingestion should be"}
	for _, pattern := range patterns {
		if containsFold(text, pattern) {
			summary := extractSummaryFromPattern(text, pattern)
			if !isUsefulSummary(summary) {
				return memories.Candidate{}, false
			}
			return candidateFromSummary(sessionFile, entry, "constraint", summary, text, 0.82, 0.90), true
		}
	}
	return memories.Candidate{}, false
}

func taskCandidate(sessionFile string, entry SessionEntry, text string) (memories.Candidate, bool) {
	patterns := []string{"next we should", "need to", "still need to", "todo", "to do"}
	for _, pattern := range patterns {
		if containsFold(text, pattern) {
			summary := extractSummaryFromPattern(text, pattern)
			if !isUsefulTaskSummary(summary) {
				return memories.Candidate{}, false
			}
			return candidateFromSummary(sessionFile, entry, "task", summary, text, 0.76, 0.62), true
		}
	}
	return memories.Candidate{}, false
}

func factCandidate(sessionFile string, entry SessionEntry, text string) (memories.Candidate, bool) {
	patterns := []string{"this project is", "we are building", "it will include", "one db per project", "one sqlite database per project", "sqlite db per project", "pi package", "typescript pi extension", "go backend", "as little tokens as possible", "algorithmic", "ingestion would be"}
	for _, pattern := range patterns {
		if containsFold(text, pattern) {
			summary := normalizeFactSummary(text)
			if summary == "" {
				summary = extractSummaryFromPattern(text, pattern)
			}
			if !isUsefulSummary(summary) {
				return memories.Candidate{}, false
			}
			return candidateFromSummary(sessionFile, entry, "fact", summary, text, 0.82, 0.78), true
		}
	}
	return memories.Candidate{}, false
}

func candidateFromSummary(sessionFile string, entry SessionEntry, category, summary, details string, confidence, importance float64) memories.Candidate {
	return memories.Candidate{
		Category:    category,
		Summary:     sentence(summary),
		Details:     details,
		SourceType:  "auto",
		Confidence:  confidence,
		Importance:  importance,
		EntryID:     entry.ID,
		EntryRole:   entry.Message.Role,
		Excerpt:     sentence(summary),
		SessionFile: sessionFile,
	}
}

func ExtractEntryText(entry SessionEntry) string {
	if entry.Message == nil {
		return ""
	}
	text := extractText(entry.Message.Content)
	return normalize(text)
}

func extractText(raw json.RawMessage) string {
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return asString
	}
	var blocks []textBlock
	if err := json.Unmarshal(raw, &blocks); err == nil {
		parts := make([]string, 0, len(blocks))
		for _, block := range blocks {
			if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
				parts = append(parts, block.Text)
			}
		}
		return strings.Join(parts, " ")
	}
	return ""
}

func normalize(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}

func sentence(text string) string {
	text = normalize(text)
	if text == "" {
		return ""
	}
	if len(text) > 160 {
		text = strings.TrimSpace(text[:160])
	}
	return strings.Trim(text, " -:;,")
}

func classifyExplicit(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "prefer") || strings.Contains(lower, "please use"):
		return "preference"
	case strings.Contains(lower, "decid") || strings.Contains(lower, "go with") || strings.Contains(lower, "we will use"):
		return "decision"
	case strings.Contains(lower, "must") || strings.Contains(lower, "never") || strings.Contains(lower, "always"):
		return "constraint"
	case strings.Contains(lower, "need to") || strings.Contains(lower, "todo"):
		return "task"
	default:
		return "fact"
	}
}

func cleanupAfterPhrase(text, phrase string) string {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, strings.ToLower(phrase))
	if idx == -1 {
		return text
	}
	cleaned := strings.TrimSpace(text[idx+len(phrase):])
	cleaned = strings.TrimLeft(cleaned, ":,.- ")
	return cleaned
}

func dedupeCandidates(candidates []memories.Candidate) []memories.Candidate {
	seen := map[string]bool{}
	result := make([]memories.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if !isUsefulSummary(candidate.Summary) {
			continue
		}
		key := candidate.Category + "|" + strings.ToLower(candidate.Summary)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, candidate)
	}
	return result
}

func containsFold(text, needle string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(needle))
}

func extractSummaryFromPattern(text, pattern string) string {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, strings.ToLower(pattern))
	if idx == -1 {
		return bestSnippet(text)
	}
	return bestSnippet(text[idx:])
}

func bestSnippet(text string) string {
	text = normalize(text)
	if text == "" {
		return ""
	}

	cutMarkers := []string{"\n", " --- ", " ## ", "### ", " - ", " 1. ", " 2. "}
	for _, marker := range cutMarkers {
		if idx := strings.Index(text, marker); idx > 0 {
			text = strings.TrimSpace(text[:idx])
			break
		}
	}

	for _, sep := range []string{". ", "? ", "! ", "; "} {
		if idx := strings.Index(text, sep); idx > 0 {
			text = strings.TrimSpace(text[:idx])
			break
		}
	}

	text = strings.Trim(text, " -:;,")
	if len(text) > 160 {
		text = strings.TrimSpace(text[:160])
	}
	return text
}

func isUsefulTaskSummary(text string) bool {
	if !isUsefulSummary(text) {
		return false
	}
	lower := strings.ToLower(normalize(text))
	weakPhrases := []string{
		"come back to me",
		"update thoughts",
		"update todos",
		"thoughts and todos",
		"check the handoff",
		"read the handoff",
		"before we continue",
		"continue where we left off",
		"keep the model sharp",
		"open ",
		"read ",
	}
	for _, phrase := range weakPhrases {
		if strings.Contains(lower, phrase) {
			return false
		}
	}
	if strings.HasPrefix(lower, "todos") || strings.HasPrefix(lower, "thoughts") {
		return false
	}
	return true
}

func isUsefulSummary(text string) bool {
	text = normalize(text)
	if text == "" {
		return false
	}
	if len(text) < 12 {
		return false
	}
	lower := strings.ToLower(text)
	if strings.Contains(lower, "##") || strings.Contains(lower, "###") {
		return false
	}
	if strings.Count(text, " - ") >= 2 {
		return false
	}
	if strings.Count(text, ":") >= 3 {
		return false
	}
	return true
}

func shouldIgnoreForMemory(text string) bool {
	lower := strings.ToLower(text)
	if len(text) > 500 {
		return true
	}
	ignorePhrases := []string{
		"check the handoff",
		"read the handoff",
		"come back to me",
		"continue where we left off",
		"keep the model sharp",
		"update thoughts and todos",
		"update our thoughts and todos",
	}
	for _, phrase := range ignorePhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	if strings.Contains(lower, "open ") && strings.Contains(lower, ".md") && len(text) < 120 {
		return true
	}
	if strings.Contains(lower, "read ") && strings.Contains(lower, ".md") && len(text) < 120 {
		return true
	}
	return false
}

func normalizeFactSummary(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "as little tokens as possible") && (strings.Contains(lower, "algorithmic") || strings.Contains(lower, "go backend")):
		return "Keep ingestion low-token and primarily algorithmic in the Go backend"
	case strings.Contains(lower, "ingestion would be") && strings.Contains(lower, "go backend"):
		return "Use the Go backend for algorithmic ingestion"
	case strings.Contains(lower, "pi package") && strings.Contains(lower, "go backend") && strings.Contains(lower, "typescript"):
		return "This project is a Pi package with a TypeScript extension and a Go backend"
	case strings.Contains(lower, "extension and the backend binary") && strings.Contains(lower, "pi package"):
		return "This project is a Pi package with a TypeScript extension and a Go backend"
	case strings.Contains(lower, "one sqlite") && strings.Contains(lower, "per project"):
		return "Use one SQLite database per project"
	case strings.Contains(lower, "sqlite db per project"):
		return "Use one SQLite database per project"
	case strings.Contains(lower, "one db per project"):
		return "Use one database per project"
	case strings.Contains(lower, "viteplus"):
		return "Use VitePlus for TypeScript workflows in this project"
	default:
		return ""
	}
}
