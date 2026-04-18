package memories

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Candidate struct {
	Category    string
	Summary     string
	Details     string
	Status      string
	SourceType  string
	Confidence  float64
	Importance  float64
	EntryID     string
	EntryRole   string
	Excerpt     string
	SessionFile string
}

type UpsertOutcome struct {
	Created bool
	Updated bool
	Ignored bool
}

var whitespaceRegexp = regexp.MustCompile(`\s+`)

func UpsertCandidate(db *sql.DB, projectID string, candidate Candidate) (*UpsertOutcome, error) {
	candidate.Summary = normalizeText(candidate.Summary)
	candidate.Details = strings.TrimSpace(candidate.Details)
	candidate.Excerpt = normalizeText(candidate.Excerpt)
	if candidate.Summary == "" || candidate.Category == "" {
		return &UpsertOutcome{Ignored: true}, nil
	}

	existingID, err := findExistingMemory(db, projectID, candidate.Category, candidate.Summary)
	if err != nil {
		return nil, err
	}
	now := timestamp()

	if existingID != "" {
		if _, err := db.Exec(`
			UPDATE memory_items
			SET details = CASE WHEN ? <> '' THEN ? ELSE details END,
			    source_type = CASE WHEN source_type = 'explicit_user' THEN source_type ELSE ? END,
			    confidence = CASE WHEN confidence < ? THEN ? ELSE confidence END,
			    importance = CASE WHEN importance < ? THEN ? ELSE importance END,
			    updated_at = ?
			WHERE memory_id = ?
		`, candidate.Details, candidate.Details, candidate.SourceType, candidate.Confidence, candidate.Confidence, candidate.Importance, candidate.Importance, now, existingID); err != nil {
			return nil, err
		}
		if strings.TrimSpace(candidate.SessionFile) != "" {
			if err := insertSource(db, existingID, candidate, now); err != nil {
				return nil, err
			}
		}
		return &UpsertOutcome{Updated: true}, nil
	}

	memoryID := "mem_" + uuid.NewString()
	if _, err := db.Exec(`
		INSERT INTO memory_items (
		  memory_id, project_id, category, summary, details, status, source_type, confidence, importance, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, memoryID, projectID, candidate.Category, candidate.Summary, candidate.Details, statusOrDefault(candidate.Status), sourceTypeOrDefault(candidate.SourceType), candidate.Confidence, candidate.Importance, now, now); err != nil {
		return nil, err
	}
	if strings.TrimSpace(candidate.SessionFile) != "" {
		if err := insertSource(db, memoryID, candidate, now); err != nil {
			return nil, err
		}
	}
	return &UpsertOutcome{Created: true}, nil
}

func findExistingMemory(db *sql.DB, projectID, category, summary string) (string, error) {
	var memoryID string
	err := db.QueryRow(`
		SELECT memory_id
		FROM memory_items
		WHERE project_id = ? AND category = ? AND lower(summary) = lower(?) AND status = 'active'
		LIMIT 1
	`, projectID, category, summary).Scan(&memoryID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return memoryID, nil
}

func insertSource(db *sql.DB, memoryID string, candidate Candidate, createdAt string) error {
	_, err := db.Exec(`
		INSERT INTO memory_sources (memory_id, session_file, entry_id, entry_role, excerpt, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, memoryID, candidate.SessionFile, nullIfEmpty(candidate.EntryID), nullIfEmpty(candidate.EntryRole), nullIfEmpty(candidate.Excerpt), createdAt)
	return err
}

func normalizeText(value string) string {
	value = strings.TrimSpace(value)
	value = whitespaceRegexp.ReplaceAllString(value, " ")
	return value
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func statusOrDefault(value string) string {
	if value == "" {
		return "active"
	}
	return value
}

func sourceTypeOrDefault(value string) string {
	if value == "" {
		return "auto"
	}
	return value
}

func timestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func DebugString(candidate Candidate) string {
	return fmt.Sprintf("[%s] %s", candidate.Category, candidate.Summary)
}
