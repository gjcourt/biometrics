# Vitals tracker (Go + SQLite)

A simple, mobile-friendly single-page web app for tracking:
- Daily weight (one entry per day; kg/lb)
- Water consumption (prominent increment button + optional decrement/undo)

Data is stored with timestamps in a local SQLite database.

## Run

From the repo root:

```bash
go run ./cmd/server
```

Then open:
- http://localhost:8080

## Configuration

Environment variables:
- `ADDR` (default `:8080`)
- `DB_PATH` (default `biometrics.sqlite`)

Example:

```bash
ADDR=:8080 DB_PATH=./biometrics.sqlite go run ./cmd/server
```

## API (quick)

- `GET /api/health`
- `GET /api/weight/today`
- `PUT /api/weight/today` body: `{ "value": 75.4, "unit": "kg" }`
- `GET /api/weight/recent?limit=14`
- `GET /api/water/today`
- `POST /api/water/event` body: `{ "deltaLiters": 0.25 }` (negative allowed)
- `GET /api/water/recent?limit=20`
- `POST /api/water/undo-last`
