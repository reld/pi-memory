package sessions

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"os"
	"sort"
	"strings"

	"pi-memory/internal/ingest"
	"pi-memory/internal/projects"
)

type SearchSessionsInput struct {
	Project    *projects.ProjectMetadata
	SessionDir string
	Query      string
	Limit      int
}

type SessionSearchRow struct {
	SessionFile string  `json:"sessionFile"`
	EntryID     string  `json:"entryId,omitempty"`
	Role        string  `json:"role,omitempty"`
	Excerpt     string  `json:"excerpt"`
	Score       float64 `json:"score"`
}

func Search(db *sql.DB, input SearchSessionsInput) ([]SessionSearchRow, error) {
	if input.Limit <= 0 {
		input.Limit = 20
	}
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return []SessionSearchRow{}, nil
	}

	if _, err := DiscoverAndTrack(db, DiscoverInput{
		Project:    input.Project,
		SessionDir: input.SessionDir,
	}); err != nil {
		return nil, err
	}

	files, err := trackedSessionFiles(db, input.Project.ProjectID)
	if err != nil {
		return nil, err
	}

	results := make([]SessionSearchRow, 0)
	for _, sessionFile := range files {
		matches, err := searchSessionFile(sessionFile, query, input.Limit)
		if err != nil {
			return nil, err
		}
		results = append(results, matches...)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			if results[i].SessionFile == results[j].SessionFile {
				return results[i].EntryID < results[j].EntryID
			}
			return results[i].SessionFile < results[j].SessionFile
		}
		return results[i].Score > results[j].Score
	})

	if len(results) > input.Limit {
		results = results[:input.Limit]
	}
	return results, nil
}

func trackedSessionFiles(db *sql.DB, projectID string) ([]string, error) {
	rows, err := db.Query(`
		SELECT session_file
		FROM tracked_sessions
		WHERE project_id = ? AND status = 'active'
		ORDER BY last_seen_at DESC, session_file ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []string
	for rows.Next() {
		var sessionFile string
		if err := rows.Scan(&sessionFile); err != nil {
			return nil, err
		}
		files = append(files, sessionFile)
	}
	return files, rows.Err()
}

func searchSessionFile(sessionFile, query string, limit int) ([]SessionSearchRow, error) {
	file, err := os.Open(sessionFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lowerQuery := strings.ToLower(strings.TrimSpace(query))
	results := make([]SessionSearchRow, 0)

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry ingest.SessionEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if entry.Message == nil {
			continue
		}
		text := ingest.ExtractEntryText(entry)
		if text == "" {
			continue
		}
		score, ok := matchScore(text, lowerQuery)
		if !ok {
			continue
		}
		results = append(results, SessionSearchRow{
			SessionFile: sessionFile,
			EntryID:     entry.ID,
			Role:        entry.Message.Role,
			Excerpt:     buildExcerpt(text, lowerQuery),
			Score:       score,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].EntryID < results[j].EntryID
		}
		return results[i].Score > results[j].Score
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func matchScore(text, lowerQuery string) (float64, bool) {
	lowerText := strings.ToLower(text)
	idx := strings.Index(lowerText, lowerQuery)
	if idx == -1 {
		return 0, false
	}
	if lowerText == lowerQuery {
		return 1.0, true
	}
	score := 0.75
	if idx == 0 || strings.Contains(lowerText, " "+lowerQuery) {
		score += 0.10
	}
	occurrences := strings.Count(lowerText, lowerQuery)
	if occurrences > 1 {
		score += 0.05
	}
	if len(text) <= 160 {
		score += 0.05
	}
	if score > 0.98 {
		score = 0.98
	}
	return score, true
}

func buildExcerpt(text, lowerQuery string) string {
	const maxLen = 220
	lowerText := strings.ToLower(text)
	idx := strings.Index(lowerText, lowerQuery)
	if idx == -1 || len(text) <= maxLen {
		return text
	}

	start := idx - 80
	if start < 0 {
		start = 0
	}
	end := idx + len(lowerQuery) + 120
	if end > len(text) {
		end = len(text)
	}

	excerpt := strings.TrimSpace(text[start:end])
	if start > 0 {
		excerpt = "…" + excerpt
	}
	if end < len(text) {
		excerpt = excerpt + "…"
	}
	return excerpt
}
