package server

import (
	"errors"
	"net/http"
	"time"
)

func (s *Server) handleChartsDaily(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	days := intQuery(r, "days", 90)
	if days > 366 {
		days = 366
	}

	unit := r.URL.Query().Get("unit")
	if unit == "" {
		unit = "lb"
	}
	if unit != "kg" && unit != "lb" {
		writeError(w, http.StatusBadRequest, errors.New("unit must be \"kg\" or \"lb\""))
		return
	}

	today := time.Now().In(time.Local)

	type weightPoint struct {
		Value float64 `json:"value"`
		Unit  string  `json:"unit"`
	}
	type dayPoint struct {
		Day         string       `json:"day"`
		WaterLiters float64      `json:"waterLiters"`
		Weight      *weightPoint `json:"weight"`
	}

	points := make([]dayPoint, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := today.AddDate(0, 0, -i)
		dayStr := localDayString(d)

		waterLiters, err := s.db.WaterTotalForLocalDay(ctx, dayStr)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		entry, err := s.db.LatestWeightForLocalDay(ctx, dayStr)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		var wp *weightPoint
		if entry != nil {
			val := entry.Value
			if entry.Unit != unit {
				val = convertWeight(val, entry.Unit, unit)
			}
			wp = &weightPoint{Value: val, Unit: unit}
		}

		points = append(points, dayPoint{Day: dayStr, WaterLiters: waterLiters, Weight: wp})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"days":  days,
		"unit":  unit,
		"today": localDayString(today),
		"items": points,
	})
}

func convertWeight(v float64, from string, to string) float64 {
	if from == to {
		return v
	}
	const kgToLb = 2.2046226218
	if from == "kg" && to == "lb" {
		return v * kgToLb
	}
	if from == "lb" && to == "kg" {
		return v / kgToLb
	}
	return v
}
