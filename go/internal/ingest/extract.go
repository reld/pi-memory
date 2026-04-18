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
	if role != "user" && role != "assistant" {
		return nil
	}
	text := ExtractEntryText(entry)
	if text == "" {
		return nil
	}

	candidates := make([]memories.Candidate, 0)
	if candidate, ok := explicitMemoryCandidate(sessionFile, entry, role, text); ok {
		candidates = append(candidates, candidate)
	}
	if role == "user" {
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
	}
	return dedupeCandidates(candidates)
}

func explicitMemoryCandidate(sessionFile string, entry SessionEntry, role, text string) (memories.Candidate, bool) {
	lower := strings.ToLower(text)
	phrases := []string{"remember this", "please remember", "note this", "remember that"}
	for _, phrase := range phrases {
		if strings.Contains(lower, phrase) {
			summary := cleanupAfterPhrase(text, phrase)
			if summary == "" {
				summary = text
			}
			return memories.Candidate{
				Category:    classifyExplicit(summary),
				Summary:     sentence(summary),
				Details:     text,
				SourceType:  "explicit_user",
				Confidence:  0.95,
				Importance:  0.90,
				EntryID:     entry.ID,
				EntryRole:   role,
				Excerpt:     text,
				SessionFile: sessionFile,
			}, true
		}
	}
	return memories.Candidate{}, false
}

func preferenceCandidate(sessionFile string, entry SessionEntry, text string) (memories.Candidate, bool) {
	lower := strings.ToLower(text)
	patterns := []string{"i prefer", "please use", "don't use", "do not use", "prefer "}
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return baseCandidate(sessionFile, entry, "preference", text, 0.75, 0.75), true
		}
	}
	return memories.Candidate{}, false
}

func decisionCandidate(sessionFile string, entry SessionEntry, text string) (memories.Candidate, bool) {
	lower := strings.ToLower(text)
	patterns := []string{"we decided", "let's go with", "we will use", "we should use", "let us use"}
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return baseCandidate(sessionFile, entry, "decision", text, 0.8, 0.85), true
		}
	}
	return memories.Candidate{}, false
}

func constraintCandidate(sessionFile string, entry SessionEntry, text string) (memories.Candidate, bool) {
	lower := strings.ToLower(text)
	patterns := []string{"must ", "never ", "always ", "keep ", "should use as few tokens as possible", "default ingestion should be"}
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return baseCandidate(sessionFile, entry, "constraint", text, 0.78, 0.88), true
		}
	}
	return memories.Candidate{}, false
}

func taskCandidate(sessionFile string, entry SessionEntry, text string) (memories.Candidate, bool) {
	lower := strings.ToLower(text)
	patterns := []string{"next we should", "need to", "still need to", "todo", "to do"}
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return baseCandidate(sessionFile, entry, "task", text, 0.72, 0.6), true
		}
	}
	return memories.Candidate{}, false
}

func factCandidate(sessionFile string, entry SessionEntry, text string) (memories.Candidate, bool) {
	lower := strings.ToLower(text)
	patterns := []string{"this project is", "we are building", "it will include", "one db per project"}
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return baseCandidate(sessionFile, entry, "fact", text, 0.7, 0.65), true
		}
	}
	return memories.Candidate{}, false
}

func baseCandidate(sessionFile string, entry SessionEntry, category, text string, confidence, importance float64) memories.Candidate {
	return memories.Candidate{
		Category:    category,
		Summary:     sentence(text),
		Details:     text,
		SourceType:  "auto",
		Confidence:  confidence,
		Importance:  importance,
		EntryID:     entry.ID,
		EntryRole:   entry.Message.Role,
		Excerpt:     text,
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
	if len(text) > 240 {
		text = strings.TrimSpace(text[:240])
	}
	return text
}

func classifyExplicit(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "prefer") || strings.Contains(lower, "please use"):
		return "preference"
	case strings.Contains(lower, "decid") || strings.Contains(lower, "go with"):
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
	idx := strings.Index(lower, phrase)
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
		key := candidate.Category + "|" + strings.ToLower(candidate.Summary)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, candidate)
	}
	return result
}
