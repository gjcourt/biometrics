package server

import (
	"errors"
	"net/http"
	"time"
)

func (s *Server) handleWeightToday(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	today := localDayString(time.Now())

	switch r.Method {
	case http.MethodGet:
		entry, err := s.db.LatestWeightForLocalDay(ctx, today)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"today": today, "entry": entry})

	case http.MethodPut:
		var body struct {
			Value float64 `json:"value"`
			Unit  string  `json:"unit"`
		}
		if err := parseJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if body.Value <= 0 {
			writeError(w, http.StatusBadRequest, errors.New("value must be > 0"))
			return
		}
		if body.Unit != "kg" && body.Unit != "lb" {
			writeError(w, http.StatusBadRequest, errors.New("unit must be \"kg\" or \"lb\""))
			return
		}
		if _, err := s.db.AddWeightEvent(ctx, body.Value, body.Unit, time.Now()); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		entry, _ := s.db.LatestWeightForLocalDay(ctx, today)
		writeJSON(w, http.StatusOK, map[string]any{"today": today, "entry": entry})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWeightRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	limit := intQuery(r, "limit", 14)
	items, err := s.db.ListRecentWeightEvents(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleWeightUndoLast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	today := localDayString(time.Now())
	deleted, err := s.db.DeleteLatestWeightEvent(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	entry, _ := s.db.LatestWeightForLocalDay(r.Context(), today)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted": deleted, "today": today, "entry": entry})
}
