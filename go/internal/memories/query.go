package memories

import "database/sql"

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
	like := "%" + query + "%"
	rows, err := db.Query(`
		SELECT memory_id, category, summary, details, status, confidence, importance, updated_at,
		  CASE
		    WHEN lower(summary) = lower(?) THEN 1.0
		    WHEN lower(summary) LIKE lower(?) THEN 0.9
		    WHEN lower(details) LIKE lower(?) THEN 0.7
		    ELSE 0.5
		  END AS score
		FROM memory_items
		WHERE project_id = ?
		  AND status = 'active'
		  AND (lower(summary) LIKE lower(?) OR lower(details) LIKE lower(?))
		ORDER BY score DESC, importance DESC, updated_at DESC
		LIMIT ?
	`, query, like, like, projectID, like, like, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemoryRowsWithScore(rows)
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
	rows, err := db.Query(`
		SELECT memory_id, category, summary, details, status, confidence, importance, updated_at,
		  (
		    importance * confidence *
		    CASE category
		      WHEN 'constraint' THEN 1.00
		      WHEN 'decision' THEN 0.95
		      WHEN 'preference' THEN 0.90
		      WHEN 'fact' THEN 0.70
		      WHEN 'task' THEN 0.60
		      ELSE 0.50
		    END
		  ) AS score
		FROM memory_items
		WHERE project_id = ?
		  AND status = 'active'
		  AND confidence >= 0.70
		  AND importance >= 0.65
		ORDER BY score DESC, importance DESC, updated_at DESC
		LIMIT ?
	`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemoryRowsWithScore(rows)
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
