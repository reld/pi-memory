package memories

import (
	"database/sql"
	"errors"
)

var ErrMemoryNotFound = errors.New("memory not found")

type ForgetResult struct {
	MemoryID  string `json:"memoryId"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updatedAt"`
}

func Forget(db *sql.DB, projectID, memoryID, mode string) (*ForgetResult, error) {
	if mode == "" {
		mode = "suppressed"
	}
	if mode != "suppressed" && mode != "forgotten" {
		mode = "suppressed"
	}

	updatedAt := timestamp()
	result, err := db.Exec(`
		UPDATE memory_items
		SET status = ?, updated_at = ?
		WHERE project_id = ? AND memory_id = ? AND status != ?
	`, mode, updatedAt, projectID, memoryID, mode)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		var updatedAt string
		err := db.QueryRow(`
			SELECT updated_at
			FROM memory_items
			WHERE project_id = ? AND memory_id = ?
		`, projectID, memoryID).Scan(&updatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemoryNotFound
		}
		if err != nil {
			return nil, err
		}
		return &ForgetResult{MemoryID: memoryID, Status: mode, UpdatedAt: updatedAt}, nil
	}

	if err := db.QueryRow(`
		SELECT updated_at
		FROM memory_items
		WHERE project_id = ? AND memory_id = ?
	`, projectID, memoryID).Scan(&updatedAt); err != nil {
		return nil, err
	}
	return &ForgetResult{MemoryID: memoryID, Status: mode, UpdatedAt: updatedAt}, nil
}
