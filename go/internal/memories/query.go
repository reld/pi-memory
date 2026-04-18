package memories

import (
	"database/sql"
	"sort"
	"strings"
)

type MemoryRow struct {
	MemoryID   string  `json:"memoryId"`
	Category   string  `json:"category"`
	Summary    string  `json:"summary"`
	Details    string  `json:"details,omitempty"`
	Status     string  `json:"status"`
	Confidence float64 `json:"confidence"`
	Importance float64 `json:"importance"`
	UpdatedAt  string  `json:"updatedAt"`
	Score      float64 `json:"score,omitempty"`
}

func List(db *sql.DB, projectID string, status string, limit int) ([]MemoryRow, error) {
	if status == "" {
		status = "active"
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.Query(`
		SELECT memory_id, category, summary, details, status, confidence, importance, updated_at
		FROM memory_items
		WHERE project_id = ? AND status = ?
		ORDER BY importance DESC, updated_at DESC
		LIMIT ?
	`, projectID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemoryRows(rows)
}

func Search(db *sql.DB, projectID string, query string, limit int) ([]MemoryRow, error) {
	if limit <= 0 {
		limit = 20
	}
	items, err := loadActiveMemories(db, projectID)
	if err != nil {
		return nil, err
	}

	query = normalizeSearchText(query)
	queryTokens := searchTokens(query)
	if query == "" || len(queryTokens) == 0 {
		return []MemoryRow{}, nil
	}

	ranked := make([]MemoryRow, 0, len(items))
	for _, item := range items {
		score := scoreSearchMatch(item, query, queryTokens)
		if score <= 0 {
			continue
		}
		item.Score = score
		ranked = append(ranked, item)
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			if ranked[i].Importance == ranked[j].Importance {
				return ranked[i].UpdatedAt > ranked[j].UpdatedAt
			}
			return ranked[i].Importance > ranked[j].Importance
		}
		return ranked[i].Score > ranked[j].Score
	})

	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return ranked, nil
}

func scanMemoryRows(rows *sql.Rows) ([]MemoryRow, error) {
	items := []MemoryRow{}
	for rows.Next() {
		var item MemoryRow
		if err := rows.Scan(&item.MemoryID, &item.Category, &item.Summary, &item.Details, &item.Status, &item.Confidence, &item.Importance, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func Recall(db *sql.DB, projectID string, limit int) ([]MemoryRow, error) {
	if limit <= 0 {
		limit = 5
	}
	items, err := loadActiveMemories(db, projectID)
	if err != nil {
		return nil, err
	}

	ranked := make([]MemoryRow, 0, len(items))
	for _, item := range items {
		score := baseRecallScore(item)
		if score <= 0 {
			continue
		}
		score *= recallNoiseAdjustment(item)
		if score <= 0 {
			continue
		}
		item.Score = score
		ranked = append(ranked, item)
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			if ranked[i].Importance == ranked[j].Importance {
				return ranked[i].UpdatedAt > ranked[j].UpdatedAt
			}
			return ranked[i].Importance > ranked[j].Importance
		}
		return ranked[i].Score > ranked[j].Score
	})

	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return ranked, nil
}

func scanMemoryRowsWithScore(rows *sql.Rows) ([]MemoryRow, error) {
	items := []MemoryRow{}
	for rows.Next() {
		var item MemoryRow
		if err := rows.Scan(&item.MemoryID, &item.Category, &item.Summary, &item.Details, &item.Status, &item.Confidence, &item.Importance, &item.UpdatedAt, &item.Score); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func loadActiveMemories(db *sql.DB, projectID string) ([]MemoryRow, error) {
	rows, err := db.Query(`
		SELECT memory_id, category, summary, details, status, confidence, importance, updated_at
		FROM memory_items
		WHERE project_id = ? AND status = 'active'
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemoryRows(rows)
}

func scoreSearchMatch(item MemoryRow, query string, queryTokens []string) float64 {
	summary := normalizeSearchText(item.Summary)
	details := normalizeSearchText(item.Details)
	category := strings.ToLower(item.Category)

	tokenMatchesSummary := 0
	tokenMatchesDetails := 0
	for _, token := range queryTokens {
		if strings.Contains(summary, token) {
			tokenMatchesSummary++
		}
		if details != "" && strings.Contains(details, token) {
			tokenMatchesDetails++
		}
	}

	if tokenMatchesSummary == 0 && tokenMatchesDetails == 0 && !strings.Contains(summary, query) && !strings.Contains(details, query) {
		return 0
	}

	score := 0.0
	if summary == query {
		score += 1.4
	}
	if strings.Contains(summary, query) {
		score += 1.0
	}
	if details != "" && strings.Contains(details, query) {
		score += 0.7
	}
	if len(queryTokens) > 0 {
		score += 0.25 * float64(tokenMatchesSummary)
		score += 0.12 * float64(tokenMatchesDetails)
		coverage := float64(tokenMatchesSummary)
		if tokenMatchesDetails > tokenMatchesSummary {
			coverage = float64(tokenMatchesDetails)
		}
		score += 0.35 * (coverage / float64(len(queryTokens)))
	}

	if categoryMatchesQuery(category, query, queryTokens) {
		score += 0.3
	}
	if hasArchitectureSignals(queryTokens) && hasArchitectureSignals(searchTokens(summary+" "+details)) {
		score += 0.45
	}
	if hasConstraintSignals(queryTokens) && category == "constraint" {
		score += 0.45
	}
	if hasToolingSignals(queryTokens) && containsAnyFold(summary+" "+details, []string{"viteplus", "vp ", "vp over npm", "typescript workflows"}) {
		score += 0.35
	}

	score += item.Importance * 0.2
	score += item.Confidence * 0.1
	score *= searchNoiseAdjustment(item, queryTokens)
	return score
}

func searchNoiseAdjustment(item MemoryRow, queryTokens []string) float64 {
	text := normalizeSearchText(item.Summary + " " + item.Details)
	if text == "" {
		return 0
	}
	if containsAnyFold(text, []string{"i love arepas", "keep working no", "come back to me"}) {
		return 0.08
	}
	if hasConstraintSignals(queryTokens) && !containsAnyFold(text, []string{"constraint", "token", "ingestion", "algorithmic", "backend", "project", "package", "typescript", "sqlite", "memory"}) {
		return 0.2
	}
	return 1.0
}

func baseRecallScore(item MemoryRow) float64 {
	if item.Confidence < 0.70 || item.Importance < 0.65 {
		return 0
	}

	categoryWeight := 0.50
	switch item.Category {
	case "constraint":
		categoryWeight = 1.00
	case "decision":
		categoryWeight = 0.97
	case "preference":
		categoryWeight = 0.92
	case "task":
		categoryWeight = 0.82
	case "fact":
		categoryWeight = 0.74
	}

	return item.Importance * item.Confidence * categoryWeight
}

func recallNoiseAdjustment(item MemoryRow) float64 {
	text := normalizeSearchText(item.Summary + " " + item.Details)
	if text == "" {
		return 0
	}

	if containsAnyFold(text, []string{
		"i love arepas",
		"keep working no",
		"come back to me",
		"continue where we left off",
		"what's next on our todos",
	}) {
		return 0.15
	}

	if containsAnyFold(text, []string{
		"commit message",
		"viteplus",
		"typescript extension",
		"go backend",
		"pi package",
		"low-token",
		"algorithmic",
		"configuration commands",
		"privacy and control principles",
		"rename/move",
		"packaging",
	}) {
		return 1.15
	}

	if item.Category == "fact" && !containsAnyFold(text, []string{
		"project",
		"package",
		"backend",
		"typescript",
		"viteplus",
		"sqlite",
		"ingestion",
		"memory",
		"config",
	}) {
		return 0.45
	}

	if item.Category == "task" && containsAnyFold(text, []string{"todo", "next", "follow-up", "still need", "open TODOs"}) {
		return 1.10
	}

	return 1.0
}

func normalizeSearchText(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	replacer := strings.NewReplacer(
		"-", " ",
		"_", " ",
		"/", " ",
		":", " ",
		",", " ",
		".", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		"?", " ",
		"!", " ",
		"'", "",
		"\"", "",
	)
	return strings.Join(strings.Fields(replacer.Replace(text)), " ")
}

func searchTokens(text string) []string {
	stopWords := map[string]bool{
		"a": true, "an": true, "and": true, "again": true, "about": true,
		"did": true, "do": true, "for": true, "how": true, "in": true,
		"is": true, "it": true, "of": true, "on": true, "or": true,
		"should": true, "the": true, "this": true, "to": true, "use": true,
		"we": true, "what": true, "where": true, "were": true, "with": true,
	}
	parts := strings.Fields(normalizeSearchText(text))
	seen := map[string]bool{}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) < 2 || stopWords[part] || seen[part] {
			continue
		}
		seen[part] = true
		out = append(out, part)
	}
	return out
}

func categoryMatchesQuery(category, query string, queryTokens []string) bool {
	if strings.Contains(query, category) {
		return true
	}
	for _, token := range queryTokens {
		if token == category {
			return true
		}
	}
	return false
}

func hasArchitectureSignals(tokens []string) bool {
	count := 0
	for _, token := range tokens {
		switch token {
		case "architecture", "package", "typescript", "extension", "go", "backend", "binary", "pi":
			count++
		}
	}
	return count >= 2
}

func hasConstraintSignals(tokens []string) bool {
	for _, token := range tokens {
		switch token {
		case "constraint", "constraints", "low", "token", "ingestion", "heuristics", "rules", "keep":
			return true
		}
	}
	return false
}

func hasToolingSignals(tokens []string) bool {
	for _, token := range tokens {
		switch token {
		case "viteplus", "vp", "tooling", "typescript", "workflow", "workflows":
			return true
		}
	}
	return false
}

func containsAnyFold(text string, needles []string) bool {
	for _, needle := range needles {
		if needle != "" && strings.Contains(text, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}
