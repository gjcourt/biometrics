package memory

import (
	"context"
	"testing"
	"time"
)

func TestWeightRepository(t *testing.T) {
	db := New()
	ctx := context.Background()

	// Add event
	now := time.Now()
	id, err := db.AddWeightEvent(ctx, 70.0, "kg", now)
	if err != nil {
		t.Fatalf("AddWeightEvent: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero ID")
	}

	// List events
	events, err := db.ListRecentWeightEvents(ctx, 10)
	if err != nil {
		t.Fatalf("ListRecentWeightEvents: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
	if events[0].Value != 70.0 {
		t.Errorf("expected 70.0, got %f", events[0].Value)
	}
	if events[0].Day == "" {
		t.Error("expected Day to be populated")
	}

	// Latest for day
	localDay := now.Format("2006-01-02")
	latest, err := db.LatestWeightForLocalDay(ctx, localDay)
	if err != nil {
		t.Fatalf("LatestWeightForLocalDay: %v", err)
	}
	if latest == nil {
		t.Error("expected latest weight, got nil")
	} else if latest.Value != 70.0 {
		t.Errorf("expected 70.0, got %f", latest.Value)
	}

	// Delete latest
	ok, err := db.DeleteLatestWeightEvent(ctx)
	if err != nil {
		t.Fatalf("DeleteLatestWeightEvent: %v", err)
	}
	if !ok {
		t.Error("expected true")
	}

	events, _ = db.ListRecentWeightEvents(ctx, 10)
	if len(events) != 0 {
		t.Error("expected 0 events")
	}
}

func TestWaterRepository(t *testing.T) {
	db := New()
	ctx := context.Background()

	now := time.Now()
	_, err := db.AddWaterEvent(ctx, 0.25, now)
	if err != nil {
		t.Fatalf("AddWaterEvent: %v", err)
	}
	_, _ = db.AddWaterEvent(ctx, 0.5, now.Add(time.Minute))

	// List
	events, err := db.ListRecentWaterEvents(ctx, 10)
	if err != nil {
		t.Fatalf("ListRecentWaterEvents: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}

	// Total for day
	localDay := now.Format("2006-01-02")
	total, err := db.WaterTotalForLocalDay(ctx, localDay)
	if err != nil {
		t.Fatalf("WaterTotalForLocalDay: %v", err)
	}
	if total != 0.75 {
		t.Errorf("expected 0.75, got %f", total)
	}
}

func TestUserRepository(t *testing.T) {
	db := New()
	ctx := context.Background()

	u, err := db.Create(ctx, "bob", "hash")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if u.Username != "bob" {
		t.Errorf("expected bob, got %s", u.Username)
	}

	u2, err := db.GetByUsername(ctx, "bob")
	if err != nil {
		t.Fatalf("GetByUsername: %v", err)
	}
	if u2 == nil || u2.ID != u.ID {
		t.Error("failed to retrieve user")
	}

	count, _ := db.Count(ctx)
	if count != 1 {
		t.Errorf("expected 1 user, got %d", count)
	}
}

func TestSessionRepository(t *testing.T) {
	db := New()
	repo := db.NewSessionRepo()
	ctx := context.Background()

	err := repo.Create(ctx, 1, "token123", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	sess, err := repo.GetByToken(ctx, "token123")
	if err != nil {
		t.Fatalf("GetByToken: %v", err)
	}
	if sess == nil {
		t.Error("expected session, got nil")
	}

	_ = repo.Delete(ctx, "token123")
	sess, _ = repo.GetByToken(ctx, "token123")
	if sess != nil {
		t.Error("expected nil (deleted)")
	}
}
