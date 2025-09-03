package store

import (
	"database/sql"
	"errors"
	"time"
)

type SQLite struct {
	db *sql.DB
}

func NewSQLite(db *sql.DB) *SQLite {
	return &SQLite{db: db}
}

func (s *SQLite) Save(code, target string) error {
	_, err := s.db.Exec(`INSERT INTO urls(code, target) VALUES(?, ?) ON CONFLICT(code) DO UPDATE SET target=excluded.target`, code, target)
	return err
}

func (s *SQLite) Get(code string) (string, error) {
	var target string
	err := s.db.QueryRow(`SELECT target FROM urls WHERE code = ?`, code).Scan(&target)
	if err != nil {
		return "", err
	}
	return target, nil
}

func (s *SQLite) InsertClick(ev ClickEvent) error {
	_, err := s.db.Exec(`INSERT INTO clicks(code, ts, ip, ua) VALUES(?, ?, ?, ?)`, ev.Code, ev.Ts.UTC(), ev.IP, ev.UA)
	return err
}

func (s *SQLite) Stats(code string) (Stats, error) {
	var out Stats
	out.Code = code

	row := s.db.QueryRow(`SELECT target FROM urls WHERE code = ?`, code)
	if err := row.Scan(&out.Target); err != nil {
		return Stats{}, err
	}

	row = s.db.QueryRow(`SELECT COUNT(*), COUNT(DISTINCT ip), COALESCE(MAX(ts), 0) FROM clicks WHERE code = ?`, code)
	var total, unique int64
	var last sql.NullTime
	if err := row.Scan(&total, &unique, &last); err != nil {
		return Stats{}, err
	}
	out.TotalClicks = total
	out.UniqueIPs = unique
	if last.Valid {
		out.LastAccess = last.Time.UTC().Format(time.RFC3339)
	}
	return out, nil
}

func (s *SQLite) TopCodes(n int) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT code FROM (
			SELECT code, COUNT(*) as c FROM clicks GROUP BY code
			UNION
			SELECT code, 0 as c FROM urls WHERE code NOT IN (SELECT DISTINCT code FROM clicks)
		)
		ORDER BY c DESC LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		res = append(res, c)
	}
	return res, rows.Err()
}

// Migrate ensures schema exists
func Migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS urls (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT UNIQUE NOT NULL,
			target TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_urls_code ON urls(code);`,
		`CREATE TABLE IF NOT EXISTS clicks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL,
			ts DATETIME DEFAULT CURRENT_TIMESTAMP,
			ip TEXT,
			ua TEXT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_clicks_code_ts ON clicks(code, ts);`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

var ErrNotFound = errors.New("not found")
