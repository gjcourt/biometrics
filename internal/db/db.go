package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	sql *sql.DB
}

func Open(path string) (*DB, error) {
	// modernc sqlite uses file: URLs; plain paths are also accepted.
	s, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// SQLite is single-writer; keep pool small.
	s.SetMaxOpenConns(1)
	s.SetMaxIdleConns(1)
	s.SetConnMaxLifetime(0)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.PingContext(ctx); err != nil {
		_ = s.Close()
		return nil, err
	}

	d := &DB{sql: s}
	if err := d.migrate(ctx); err != nil {
		_ = s.Close()
		return nil, err
	}
	return d, nil
}

func (d *DB) Close() error {
	return d.sql.Close()
}

func (d *DB) migrate(ctx context.Context) error {
	stmts := []string{
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS weights (
			day TEXT PRIMARY KEY, -- YYYY-MM-DD in local time
			value REAL NOT NULL,
			unit TEXT NOT NULL CHECK(unit IN ('kg','lb')),
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS weight_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			value REAL NOT NULL,
			unit TEXT NOT NULL CHECK(unit IN ('kg','lb')),
			created_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_weight_events_created_at ON weight_events(created_at);`,
		`CREATE TABLE IF NOT EXISTS water_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			delta_liters REAL NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_water_events_created_at ON water_events(created_at);`,
	}

	for _, stmt := range stmts {
		if _, err := d.sql.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	// One-time migration: if we have legacy per-day weights but no events yet,
	// copy those rows into the event table.
	var eventCount int
	if err := d.sql.QueryRowContext(ctx, `SELECT COUNT(1) FROM weight_events;`).Scan(&eventCount); err != nil {
		return fmt.Errorf("migrate: count weight_events: %w", err)
	}
	if eventCount == 0 {
		if _, err := d.sql.ExecContext(ctx, `INSERT INTO weight_events(value, unit, created_at) SELECT value, unit, created_at FROM weights;`); err != nil {
			return fmt.Errorf("migrate: migrate weights->weight_events: %w", err)
		}
	}
	return nil
}

type WeightEntry struct {
	ID        int64     `json:"id"`
	Day       string    `json:"day"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"` // kg|lb
	CreatedAt time.Time `json:"createdAt"`
}

type WaterEvent struct {
	ID          int64     `json:"id"`
	DeltaLiters float64   `json:"deltaLiters"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (d *DB) AddWeightEvent(ctx context.Context, value float64, unit string, createdAt time.Time) (int64, error) {
	res, err := d.sql.ExecContext(ctx, `INSERT INTO weight_events(value, unit, created_at) VALUES(?, ?, ?);`, value, unit, createdAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// DeleteLatestWeightEvent removes the most recent weight event (by created_at).
// Returns (deleted, nil) if successful.
func (d *DB) DeleteLatestWeightEvent(ctx context.Context) (bool, error) {
	row := d.sql.QueryRowContext(ctx, `SELECT id FROM weight_events ORDER BY created_at DESC LIMIT 1;`)
	var id int64
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	_, err := d.sql.ExecContext(ctx, `DELETE FROM weight_events WHERE id=?;`, id)
	if err != nil {
		return false, err
	}
	return true, nil
}

// LatestWeightForLocalDay returns the most recently recorded weight for the given local day (YYYY-MM-DD).
func (d *DB) LatestWeightForLocalDay(ctx context.Context, localDay string) (*WeightEntry, error) {
	dayStartLocal, err := time.ParseInLocation("2006-01-02", localDay, time.Local)
	if err != nil {
		return nil, err
	}
	dayEndLocal := dayStartLocal.Add(24 * time.Hour)

	startUTC := dayStartLocal.UTC().Format(time.RFC3339Nano)
	endUTC := dayEndLocal.UTC().Format(time.RFC3339Nano)

	row := d.sql.QueryRowContext(ctx,
		`SELECT id, value, unit, created_at FROM weight_events WHERE created_at >= ? AND created_at < ? ORDER BY created_at DESC LIMIT 1;`,
		startUTC, endUTC,
	)

	var e WeightEntry
	var created string
	if err := row.Scan(&e.ID, &e.Value, &e.Unit, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return nil, err
	}
	e.CreatedAt = t
	e.Day = localDay
	return &e, nil
}

func (d *DB) ListRecentWeightEvents(ctx context.Context, limit int) ([]WeightEntry, error) {
	rows, err := d.sql.QueryContext(ctx, `SELECT id, value, unit, created_at FROM weight_events ORDER BY created_at DESC LIMIT ?;`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]WeightEntry, 0, limit)
	for rows.Next() {
		var e WeightEntry
		var created string
		if err := rows.Scan(&e.ID, &e.Value, &e.Unit, &created); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, err
		}
		e.CreatedAt = t
		e.Day = t.In(time.Local).Format("2006-01-02")
		out = append(out, e)
	}
	return out, rows.Err()
}

func (d *DB) AddWaterEvent(ctx context.Context, deltaLiters float64, createdAt time.Time) (int64, error) {
	res, err := d.sql.ExecContext(ctx, `INSERT INTO water_events(delta_liters, created_at) VALUES(?, ?);`, deltaLiters, createdAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (d *DB) DeleteWaterEvent(ctx context.Context, id int64) error {
	_, err := d.sql.ExecContext(ctx, `DELETE FROM water_events WHERE id=?;`, id)
	return err
}

func (d *DB) ListRecentWaterEvents(ctx context.Context, limit int) ([]WaterEvent, error) {
	rows, err := d.sql.QueryContext(ctx, `SELECT id, delta_liters, created_at FROM water_events ORDER BY created_at DESC LIMIT ?;`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]WaterEvent, 0, limit)
	for rows.Next() {
		var e WaterEvent
		var created string
		if err := rows.Scan(&e.ID, &e.DeltaLiters, &created); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, err
		}
		e.CreatedAt = t
		out = append(out, e)
	}
	return out, rows.Err()
}

// WaterTotalForLocalDay returns the total water for the given local day (YYYY-MM-DD),
// while timestamps are stored in UTC.
func (d *DB) WaterTotalForLocalDay(ctx context.Context, localDay string) (float64, error) {
	dayStartLocal, err := time.ParseInLocation("2006-01-02", localDay, time.Local)
	if err != nil {
		return 0, err
	}
	dayEndLocal := dayStartLocal.Add(24 * time.Hour)

	startUTC := dayStartLocal.UTC().Format(time.RFC3339Nano)
	endUTC := dayEndLocal.UTC().Format(time.RFC3339Nano)

	row := d.sql.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(delta_liters), 0) FROM water_events WHERE created_at >= ? AND created_at < ?;`,
		startUTC, endUTC,
	)
	var total float64
	if err := row.Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}
