package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS alerts (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    alert_id        TEXT NOT NULL UNIQUE,
    company_id      INTEGER NOT NULL,
    sub_account_id  INTEGER NOT NULL,
    sent            INTEGER NOT NULL DEFAULT 0,
    bounced         INTEGER NOT NULL DEFAULT 0,
    spam_bounced    INTEGER NOT NULL DEFAULT 0,
    bounce_rate_pct REAL NOT NULL DEFAULT 0,
    spam_rate_pct   REAL NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'detected',
    transmission_id TEXT,
    detected_at     DATETIME NOT NULL,
    resolved_at     DATETIME
);

CREATE INDEX IF NOT EXISTS idx_alerts_company_detected ON alerts(company_id, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_detected_at ON alerts(detected_at DESC);
`

func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	return db, nil
}
