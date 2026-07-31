package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type CheckRecord struct {
	ID         int64  `json:"id"`
	SweepID    int    `json:"sweep_id"`
	Timestamp  string `json:"timestamp"`
	Target     string `json:"target"`
	Status     string `json:"status"`
	StatusCode int    `json:"status_code"`
	LatencyMs  int64  `json:"latency_ms"`
	Error      string `json:"error"`
}

type UptimeStats struct {
	Target       string  `json:"target"`
	TotalChecks  int     `json:"total_checks"`
	UpChecks     int     `json:"up_checks"`
	DownChecks   int     `json:"down_checks"`
	UptimePct    float64 `json:"uptime_pct"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)

	if err := createSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func createSchema(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS checks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sweep_id INTEGER NOT NULL,
		timestamp TEXT NOT NULL,
		target TEXT NOT NULL,
		status TEXT NOT NULL,
		status_code INTEGER DEFAULT 0,
		latency_ms INTEGER DEFAULT 0,
		error TEXT DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_checks_target_timestamp ON checks(target, timestamp);
	`
	_, err := db.Exec(query)
	return err
}

func (s *Store) Record(sweepID int, timestamp time.Time, target, status string, statusCode int, latencyMs int64, errMsg string) error {
	query := `INSERT INTO checks (sweep_id, timestamp, target, status, status_code, latency_ms, error) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, dbErr := s.db.Exec(query,
		sweepID,
		timestamp.UTC().Format(time.RFC3339),
		target,
		status,
		statusCode,
		latencyMs,
		errMsg,
	)
	return dbErr
}

func (s *Store) GetHistory(target string, limit int) ([]CheckRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows *sql.Rows
	var err error
	if target == "" {
		rows, err = s.db.Query(`SELECT id, sweep_id, timestamp, target, status, status_code, latency_ms, error FROM checks ORDER BY id DESC LIMIT ?`, limit)
	} else {
		rows, err = s.db.Query(`SELECT id, sweep_id, timestamp, target, status, status_code, latency_ms, error FROM checks WHERE target = ? ORDER BY id DESC LIMIT ?`, target, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []CheckRecord
	for rows.Next() {
		var r CheckRecord
		if err := rows.Scan(&r.ID, &r.SweepID, &r.Timestamp, &r.Target, &r.Status, &r.StatusCode, &r.LatencyMs, &r.Error); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func (s *Store) GetStats(target string) (UptimeStats, error) {
	query := `SELECT target, COUNT(*), SUM(CASE WHEN status='UP' THEN 1 ELSE 0 END), SUM(CASE WHEN status='DOWN' THEN 1 ELSE 0 END), COALESCE(AVG(latency_ms), 0) FROM checks WHERE target = ?`
	row := s.db.QueryRow(query, target)

	var stats UptimeStats
	var avgLat float64
	if err := row.Scan(&stats.Target, &stats.TotalChecks, &stats.UpChecks, &stats.DownChecks, &avgLat); err != nil {
		return stats, err
	}
	stats.AvgLatencyMs = avgLat
	if stats.TotalChecks > 0 {
		stats.UptimePct = float64(stats.UpChecks) / float64(stats.TotalChecks) * 100
	}
	return stats, nil
}

func (s *Store) GetAllStats() ([]UptimeStats, error) {
	rows, err := s.db.Query(`SELECT DISTINCT target FROM checks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []UptimeStats
	for rows.Next() {
		var target string
		if err := rows.Scan(&target); err != nil {
			return nil, err
		}
		stats, err := s.GetStats(target)
		if err != nil {
			continue
		}
		results = append(results, stats)
	}
	return results, rows.Err()
}

func (s *Store) Close() error {
	return s.db.Close()
}
