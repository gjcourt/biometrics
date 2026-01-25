package server

import (
	"errors"
	"net/http"
	"time"
)

func (s *Server) handleWaterToday(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	today := localDayString(time.Now())
	total, err := s.db.WaterTotalForLocalDay(r.Context(), today)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"today": today, "totalLiters": total})
}

func (s *Server) handleWaterEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		DeltaLiters float64 `json:"deltaLiters"`
	}
	if err := parseJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if body.DeltaLiters == 0 || body.DeltaLiters < -10 || body.DeltaLiters > 10 {
		writeError(w, http.StatusBadRequest, errors.New("deltaLiters must be non-zero and within [-10, 10]"))
		return
	}
	id, err := s.db.AddWaterEvent(r.Context(), body.DeltaLiters, time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id})
}

func (s *Server) handleWaterRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	limit := intQuery(r, "limit", 20)
	items, err := s.db.ListRecentWaterEvents(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleWaterUndoLast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	items, err := s.db.ListRecentWaterEvents(r.Context(), 1)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(items) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"undone": false})
		return
	}
	if err := s.db.DeleteWaterEvent(r.Context(), items[0].ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"undone": true, "id": items[0].ID})
}
