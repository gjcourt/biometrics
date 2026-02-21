# Vitals

A simple, mobile-friendly web app for tracking daily weight and water intake.
Built with Go, PostgreSQL, and vanilla JS.

## Documentation

For detailed documentation, please refer to the [docs/](./docs/) folder:
- [API Documentation](./docs/api.md)
- [Authentication](./docs/authentication.md)
- [Database Schema](./docs/database.md)
- [Architecture](./docs/architecture.md)

## Architecture

The project follows **hexagonal (ports & adapters) architecture**:

```
cmd/vitals/          ← entry point, wires everything together
internal/
  domain/                ← core: entities + port interfaces (zero external deps)
  app/                   ← application services (business logic + validation)
  adapter/
    postgres/            ← driven adapter: implements domain repository ports
    http/                ← driving adapter: HTTP handlers calling app services
web/                     ← static frontend assets (HTML/CSS/JS)
```

## Build & Run

```bash
# Build
go build ./...

# Test
go test ./...

# Run locally (In-Memory)
go run ./cmd/vitals

# Run locally (PostgreSQL)
POSTGRES_URL="postgres://user:pass@localhost:5432/vitals?sslmode=disable" \
  go run ./cmd/vitals

# Docker
docker build -t vitals .
docker run -e POSTGRES_URL="..." -p 8080:8080 vitals
```

Then open http://localhost:8080

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `POSTGRES_URL` | *(optional)* | PostgreSQL connection string. If unset, uses in-memory DB. |
| `POSTGRES_USER` | *(optional)* | Override user for Postgres connection (maps to PGUSER). |
| `POSTGRES_PASSWORD` | *(optional)* | Override password for Postgres connection (maps to PGPASSWORD). |
| `ADDR` | `:8080` | Listen address |
| `WEB_DIR` | `web` | Path to static frontend assets |

## API

- `GET /api/health`
- `GET /api/weight/today`
- `PUT /api/weight/today` — body: `{ "value": 75.4, "unit": "kg" }`
- `GET /api/weight/recent?limit=14`
- `POST /api/weight/undo-last`
- `GET /api/water/today`
- `POST /api/water/event` — body: `{ "deltaLiters": 0.25 }`
- `GET /api/water/recent?limit=20`
- `POST /api/water/undo-last`
- `GET /api/charts/daily?days=90&unit=lb`
