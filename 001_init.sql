-- Initial schema (also applied programmatically in Migrate)
CREATE TABLE IF NOT EXISTS urls (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT UNIQUE NOT NULL,
  target TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_urls_code ON urls(code);

CREATE TABLE IF NOT EXISTS clicks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL,
  ts DATETIME DEFAULT CURRENT_TIMESTAMP,
  ip TEXT,
  ua TEXT
);
CREATE INDEX IF NOT EXISTS idx_clicks_code_ts ON clicks(code, ts);
