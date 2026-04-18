package migrations

import "database/sql"

var statements = []string{
	`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);`,
	`CREATE TABLE IF NOT EXISTS tracked_sessions (
	  session_file TEXT PRIMARY KEY,
	  project_id TEXT NOT NULL,
	  session_id TEXT,
	  session_name TEXT,
	  first_seen_at TEXT NOT NULL,
	  last_seen_at TEXT NOT NULL,
	  last_ingested_at TEXT,
	  status TEXT NOT NULL CHECK (status IN ('active', 'missing', 'stale'))
	);`,
	`CREATE INDEX IF NOT EXISTS idx_tracked_sessions_project_id ON tracked_sessions(project_id);`,
	`CREATE INDEX IF NOT EXISTS idx_tracked_sessions_status ON tracked_sessions(status);`,
	`CREATE INDEX IF NOT EXISTS idx_tracked_sessions_last_ingested_at ON tracked_sessions(last_ingested_at);`,
	`CREATE TABLE IF NOT EXISTS memory_items (
	  memory_id TEXT PRIMARY KEY,
	  project_id TEXT NOT NULL,
	  category TEXT NOT NULL CHECK (category IN ('preference', 'fact', 'decision', 'task', 'constraint')),
	  summary TEXT NOT NULL,
	  details TEXT,
	  status TEXT NOT NULL CHECK (status IN ('active', 'suppressed', 'forgotten')),
	  source_type TEXT NOT NULL CHECK (source_type IN ('auto', 'explicit_user')),
	  confidence REAL NOT NULL CHECK (confidence >= 0.0 AND confidence <= 1.0),
	  importance REAL NOT NULL CHECK (importance >= 0.0 AND importance <= 1.0),
	  created_at TEXT NOT NULL,
	  updated_at TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_memory_items_project_id ON memory_items(project_id);`,
	`CREATE INDEX IF NOT EXISTS idx_memory_items_category ON memory_items(category);`,
	`CREATE INDEX IF NOT EXISTS idx_memory_items_status ON memory_items(status);`,
	`CREATE INDEX IF NOT EXISTS idx_memory_items_source_type ON memory_items(source_type);`,
	`CREATE INDEX IF NOT EXISTS idx_memory_items_updated_at ON memory_items(updated_at DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_memory_items_importance ON memory_items(importance DESC);`,
	`CREATE TABLE IF NOT EXISTS memory_sources (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  memory_id TEXT NOT NULL,
	  session_file TEXT NOT NULL,
	  entry_id TEXT,
	  entry_role TEXT,
	  excerpt TEXT,
	  created_at TEXT NOT NULL,
	  FOREIGN KEY (memory_id) REFERENCES memory_items(memory_id) ON DELETE CASCADE,
	  FOREIGN KEY (session_file) REFERENCES tracked_sessions(session_file) ON DELETE CASCADE
	);`,
	`CREATE INDEX IF NOT EXISTS idx_memory_sources_memory_id ON memory_sources(memory_id);`,
	`CREATE INDEX IF NOT EXISTS idx_memory_sources_session_file ON memory_sources(session_file);`,
	`CREATE INDEX IF NOT EXISTS idx_memory_sources_entry_id ON memory_sources(entry_id);`,
	`CREATE TABLE IF NOT EXISTS ingestion_runs (
	  run_id TEXT PRIMARY KEY,
	  started_at TEXT NOT NULL,
	  finished_at TEXT,
	  status TEXT NOT NULL CHECK (status IN ('running', 'completed', 'failed')),
	  trigger TEXT NOT NULL CHECK (trigger IN ('auto_turn', 'session_start_catchup', 'manual', 'explicit_user')),
	  session_file TEXT,
	  entries_seen INTEGER NOT NULL DEFAULT 0,
	  candidates_found INTEGER NOT NULL DEFAULT 0,
	  memories_created INTEGER NOT NULL DEFAULT 0,
	  memories_updated INTEGER NOT NULL DEFAULT 0,
	  memories_ignored INTEGER NOT NULL DEFAULT 0,
	  error_message TEXT,
	  FOREIGN KEY (session_file) REFERENCES tracked_sessions(session_file) ON DELETE SET NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_ingestion_runs_status ON ingestion_runs(status);`,
	`CREATE INDEX IF NOT EXISTS idx_ingestion_runs_trigger ON ingestion_runs(trigger);`,
	`CREATE INDEX IF NOT EXISTS idx_ingestion_runs_session_file ON ingestion_runs(session_file);`,
	`CREATE INDEX IF NOT EXISTS idx_ingestion_runs_started_at ON ingestion_runs(started_at DESC);`,
	`CREATE TABLE IF NOT EXISTS ingestion_state (
	  session_file TEXT PRIMARY KEY,
	  last_entry_id TEXT,
	  last_entry_timestamp TEXT,
	  last_ingested_at TEXT NOT NULL,
	  last_run_id TEXT,
	  FOREIGN KEY (session_file) REFERENCES tracked_sessions(session_file) ON DELETE CASCADE,
	  FOREIGN KEY (last_run_id) REFERENCES ingestion_runs(run_id) ON DELETE SET NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_ingestion_state_last_ingested_at ON ingestion_state(last_ingested_at DESC);`,
	`CREATE TABLE IF NOT EXISTS memory_tags (
	  memory_id TEXT NOT NULL,
	  tag TEXT NOT NULL,
	  PRIMARY KEY (memory_id, tag),
	  FOREIGN KEY (memory_id) REFERENCES memory_items(memory_id) ON DELETE CASCADE
	);`,
}

func Apply(db *sql.DB) error {
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
