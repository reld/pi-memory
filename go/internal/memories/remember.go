package memories

import (
	"database/sql"
	"strings"
)

type RememberResult struct {
	MemoryID   string  `json:"memoryId"`
	Category   string  `json:"category"`
	Summary    string  `json:"summary"`
	Status     string  `json:"status"`
	Confidence float64 `json:"confidence"`
	Importance float64 `json:"importance"`
	UpdatedAt  string  `json:"updatedAt"`
	Created    bool    `json:"created"`
}

func Remember(db *sql.DB, projectID, text string) (*RememberResult, error) {
	category := classifyRememberText(text)
	candidate := Candidate{
		Category:    category,
		Summary:     summarizeRememberText(text),
		Details:     normalizeText(text),
		SourceType:  "explicit_user",
		Confidence:  0.95,
		Importance:  0.90,
		Status:      "active",
		EntryRole:   "user",
		Excerpt:     normalizeText(text),
		SessionFile: "",
	}

	existingID, err := findExistingMemory(db, projectID, candidate.Category, normalizeText(candidate.Summary))
	if err != nil {
		return nil, err
	}

	outcome, err := UpsertCandidate(db, projectID, candidate)
	if err != nil {
		return nil, err
	}

	memoryID := existingID
	if memoryID == "" {
		err := db.QueryRow(`
			SELECT memory_id
			FROM memory_items
			WHERE project_id = ? AND category = ? AND lower(summary) = lower(?)
			ORDER BY updated_at DESC
			LIMIT 1
		`, projectID, candidate.Category, candidate.Summary).Scan(&memoryID)
		if err != nil {
			return nil, err
		}
	}

	var result RememberResult
	result.Created = outcome.Created
	if err := db.QueryRow(`
		SELECT memory_id, category, summary, status, confidence, importance, updated_at
		FROM memory_items
		WHERE memory_id = ?
	`, memoryID).Scan(&result.MemoryID, &result.Category, &result.Summary, &result.Status, &result.Confidence, &result.Importance, &result.UpdatedAt); err != nil {
		return nil, err
	}
	return &result, nil
}

func classifyRememberText(text string) string {
	lower := normalizeText(text)
	switch {
	case containsAny(lower, []string{"prefer", "please use", "don't use", "do not use"}):
		return "preference"
	case containsAny(lower, []string{"decid", "go with", "we will use", "should use"}):
		return "decision"
	case containsAny(lower, []string{"must", "never", "always", "constraint"}):
		return "constraint"
	case containsAny(lower, []string{"need to", "todo", "next step", "follow up"}):
		return "task"
	default:
		return "fact"
	}
}

func summarizeRememberText(text string) string {
	text = normalizeText(text)
	if len(text) > 240 {
		return text[:240]
	}
	return text
}

func containsAny(text string, needles []string) bool {
	lower := strings.ToLower(text)
	for _, needle := range needles {
		if needle != "" && strings.Contains(lower, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}
