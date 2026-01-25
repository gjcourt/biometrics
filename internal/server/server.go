package server

import (
	"net/http"

	"biometrics/internal/db"
)

type Server struct {
	db     *db.DB
	webDir string
}

func New(database *db.DB, webDir string) *Server {
	return &Server{db: database, webDir: webDir}
}

func (s *Server) Handler() http.Handler {
	api := http.NewServeMux()
	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	api.HandleFunc("/weight/today", s.handleWeightToday)
	api.HandleFunc("/weight/recent", s.handleWeightRecent)
	api.HandleFunc("/weight/undo-last", s.handleWeightUndoLast)

	api.HandleFunc("/water/today", s.handleWaterToday)
	api.HandleFunc("/water/event", s.handleWaterEvent)
	api.HandleFunc("/water/recent", s.handleWaterRecent)
	api.HandleFunc("/water/undo-last", s.handleWaterUndoLast)

	api.HandleFunc("/charts/daily", s.handleChartsDaily)

	root := http.NewServeMux()
	root.Handle("/api/", http.StripPrefix("/api", api))
	root.Handle("/", spaFromDisk(s.webDir))

	return withNoCache(root)
}
