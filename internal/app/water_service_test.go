package app_test

import (
	"context"
	"testing"
	"time"

	"biometrics/internal/app"
	"biometrics/internal/domain"
)

type mockWaterRepo struct {
	addFn   func(ctx context.Context, d float64, t time.Time) (int64, error)
	delFn   func(ctx context.Context, id int64) error
	listFn  func(ctx context.Context, limit int) ([]domain.WaterEvent, error)
	totalFn func(ctx context.Context, day string) (float64, error)
}

func (m *mockWaterRepo) AddWaterEvent(ctx context.Context, d float64, t time.Time) (int64, error) {
	if m.addFn != nil {
		return m.addFn(ctx, d, t)
	}
	return 0, nil
}

func (m *mockWaterRepo) DeleteWaterEvent(ctx context.Context, id int64) error {
	if m.delFn != nil {
		return m.delFn(ctx, id)
	}
	return nil
}

func (m *mockWaterRepo) ListRecentWaterEvents(ctx context.Context, limit int) ([]domain.WaterEvent, error) {
	if m.listFn != nil {
		return m.listFn(ctx, limit)
	}
	return nil, nil
}

func (m *mockWaterRepo) WaterTotalForLocalDay(ctx context.Context, day string) (float64, error) {
	if m.totalFn != nil {
		return m.totalFn(ctx, day)
	}
	return 0, nil
}

func TestRecordWaterEvent_Validation(t *testing.T) {
	svc := app.NewWaterService(&mockWaterRepo{})

	tests := []struct {
		name  string
		delta float64
	}{
		{"zero delta", 0},
		{"too large positive", 15},
		{"too large negative", -15},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.RecordEvent(context.Background(), tc.delta)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestRecordWaterEvent_Success(t *testing.T) {
	repo := &mockWaterRepo{
		addFn: func(_ context.Context, _ float64, _ time.Time) (int64, error) { return 42, nil },
	}
	svc := app.NewWaterService(repo)
	id, err := svc.RecordEvent(context.Background(), 0.25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Fatalf("expected id 42, got %d", id)
	}
}

func TestUndoLastWater_Empty(t *testing.T) {
	repo := &mockWaterRepo{
		listFn: func(_ context.Context, _ int) ([]domain.WaterEvent, error) {
			return nil, nil
		},
	}
	svc := app.NewWaterService(repo)
	undone, _, err := svc.UndoLast(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if undone {
		t.Fatal("expected undone=false for empty list")
	}
}

func TestUndoLastWater_Success(t *testing.T) {
	repo := &mockWaterRepo{
		listFn: func(_ context.Context, _ int) ([]domain.WaterEvent, error) {
			return []domain.WaterEvent{{ID: 7, DeltaLiters: 0.5}}, nil
		},
		delFn: func(_ context.Context, id int64) error {
			if id != 7 {
				t.Fatalf("expected delete id 7, got %d", id)
			}
			return nil
		},
	}
	svc := app.NewWaterService(repo)
	undone, id, err := svc.UndoLast(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !undone || id != 7 {
		t.Fatalf("expected undone=true id=7, got undone=%v id=%d", undone, id)
	}
}

func TestGetTodayTotal(t *testing.T) {
	repo := &mockWaterRepo{
		totalFn: func(_ context.Context, day string) (float64, error) {
			if day != "2026-02-08" {
				t.Fatalf("unexpected day: %s", day)
			}
			return 2.5, nil
		},
	}
	svc := app.NewWaterService(repo)
	total, err := svc.GetTodayTotal(context.Background(), "2026-02-08")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2.5 {
		t.Fatalf("expected 2.5, got %v", total)
	}
}
