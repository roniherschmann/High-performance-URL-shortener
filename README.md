# High-Performance URL Shortener (Go)

A practical, production-style URL shortener written in Go. It demonstrates:
- Concurrency and performance on the hot redirect path
- Async click analytics via buffered channels
- Read-through caching with `sync.Map`
- Prometheus metrics and structured JSON logs
- SQLite persistence (easily swappable to Postgres)
- Dockerized deployment

## Quick Start
```bash
# Run locally
go run ./cmd/server

# Or with Docker
docker compose up --build
