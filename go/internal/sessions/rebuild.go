package sessions

import "database/sql"

type RebuildResult struct {
	ClearedMemorySources  int           `json:"clearedMemorySources"`
	ClearedMemoryItems    int           `json:"clearedMemoryItems"`
	ClearedIngestionState int           `json:"clearedIngestionState"`
	ClearedIngestionRuns  int           `json:"clearedIngestionRuns"`
	Ingest                *IngestResult `json:"ingest"`
}

func Rebuild(db *sql.DB, input IngestInput) (*RebuildResult, error) {
	clearedSources, clearedItems, clearedState, clearedRuns, err := resetDerivedState(db, input.Project.ProjectID)
	if err != nil {
		return nil, err
	}

	result, err := Ingest(db, input)
	if err != nil {
		return nil, err
	}

	return &RebuildResult{
		ClearedMemorySources:  clearedSources,
		ClearedMemoryItems:    clearedItems,
		ClearedIngestionState: clearedState,
		ClearedIngestionRuns:  clearedRuns,
		Ingest:                result,
	}, nil
}

func resetDerivedState(db *sql.DB, projectID string) (clearedSources int, clearedItems int, clearedState int, clearedRuns int, err error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if clearedSources, err = execDeleteCount(tx, `
		DELETE FROM memory_sources
		WHERE memory_id IN (
			SELECT memory_id FROM memory_items WHERE project_id = ?
		)
	`, projectID); err != nil {
		return 0, 0, 0, 0, err
	}
	if clearedItems, err = execDeleteCount(tx, `DELETE FROM memory_items WHERE project_id = ?`, projectID); err != nil {
		return 0, 0, 0, 0, err
	}
	if clearedState, err = execDeleteCount(tx, `
		DELETE FROM ingestion_state
		WHERE session_file IN (
			SELECT session_file FROM tracked_sessions WHERE project_id = ?
		)
	`, projectID); err != nil {
		return 0, 0, 0, 0, err
	}
	if clearedRuns, err = execDeleteCount(tx, `DELETE FROM ingestion_runs`, nil); err != nil {
		return 0, 0, 0, 0, err
	}
	if _, err = tx.Exec(`UPDATE tracked_sessions SET last_ingested_at = NULL WHERE project_id = ?`, projectID); err != nil {
		return 0, 0, 0, 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, 0, 0, 0, err
	}
	return clearedSources, clearedItems, clearedState, clearedRuns, nil
}

func execDeleteCount(tx *sql.Tx, query string, arg any) (int, error) {
	var (
		res sql.Result
		err error
	)
	if arg == nil {
		res, err = tx.Exec(query)
	} else {
		res, err = tx.Exec(query, arg)
	}
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(affected), nil
}
