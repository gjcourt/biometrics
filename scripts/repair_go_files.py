from __future__ import annotations

from pathlib import Path


def write(rel_path: str, content: str) -> None:
    p = Path(__file__).resolve().parent.parent / rel_path
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(content, encoding="utf-8")


def main() -> None:
    ignored = "//go:build ignore\n// +build ignore\n\npackage main\n"

    write("cmd/app/main.go", ignored)
    write("cmd/server/main.go", ignored)
    write("cmd/server/server_main.go", ignored)

    write(
        "cmd/biometrics/main.go",
        """package main

import (
    \"errors\"
    \"log\"
    \"net/http\"
    \"os\"

    \"biometrics/internal/db\"
    \"biometrics/internal/server\"
)

func main() {
    addr := env(\"ADDR\", \":8080\")
    dbPath := env(\"DB_PATH\", \"biometrics.sqlite\")
    webDir := env(\"WEB_DIR\", \"web\")

    d, err := db.Open(dbPath)
    if err != nil {
        log.Fatalf(\"db open: %v\", err)
    }
    defer func() { _ = d.Close() }()

    h := server.New(d, webDir).Handler()
    log.Printf(\"listening on %s (db: %s)\", addr, dbPath)
    if err := http.ListenAndServe(addr, h); err != nil && !errors.Is(err, http.ErrServerClosed) {
        log.Fatal(err)
    }
}

func env(key, fallback string) string {
    if v := os.Getenv(key); v != \"\" {
        return v
    }
    return fallback
}
""",
    )

    write(
        "internal/server/server.go",
        """package server

import (
    \"net/http\"

    \"biometrics/internal/db\"
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
    api.HandleFunc(\"/health\", func(w http.ResponseWriter, r *http.Request) {
        writeJSON(w, http.StatusOK, map[string]any{\"ok\": true})
    })

    api.HandleFunc(\"/weight/today\", s.handleWeightToday)
    api.HandleFunc(\"/weight/recent\", s.handleWeightRecent)

    api.HandleFunc(\"/water/today\", s.handleWaterToday)
    api.HandleFunc(\"/water/event\", s.handleWaterEvent)
    api.HandleFunc(\"/water/recent\", s.handleWaterRecent)
    api.HandleFunc(\"/water/undo-last\", s.handleWaterUndoLast)

    root := http.NewServeMux()
    root.Handle(\"/api/\", http.StripPrefix(\"/api\", api))
    root.Handle(\"/\", spaFromDisk(s.webDir))

    return withNoCache(root)
}
""",
    )

    write(
        "internal/server/util.go",
        """package server

import (
    \"encoding/json\"
    \"fmt\"
    \"net/http\"
    \"os\"
    \"path\"
    \"strconv\"
    \"time\"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set(\"Content-Type\", \"application/json; charset=utf-8\")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
    writeJSON(w, status, map[string]any{\"error\": err.Error()})
}

func parseJSON(r *http.Request, dst any) error {
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()
    if err := dec.Decode(dst); err != nil {
        return fmt.Errorf(\"invalid json: %w\", err)
    }
    return nil
}

func intQuery(r *http.Request, key string, fallback int) int {
    v := r.URL.Query().Get(key)
    if v == \"\" {
        return fallback
    }
    n, err := strconv.Atoi(v)
    if err != nil || n <= 0 {
        return fallback
    }
    return n
}

func localDayString(t time.Time) string {
    return t.In(time.Local).Format(\"2006-01-02\")
}

func withNoCache(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set(\"Cache-Control\", \"no-store\")
        next.ServeHTTP(w, r)
    })
}

func spaFromDisk(dir string) http.Handler {
    fileServer := http.FileServer(http.Dir(dir))
    indexPath := path.Join(dir, \"index.html\")

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        reqPath := path.Clean(r.URL.Path)
        if reqPath == \"/\" {
            http.ServeFile(w, r, indexPath)
            return
        }

        staticPath := path.Join(dir, reqPath)
        if _, err := os.Stat(staticPath); err == nil {
            fileServer.ServeHTTP(w, r)
            return
        }

        http.ServeFile(w, r, indexPath)
    })
}
""",
    )

    write(
        "internal/server/handlers_weight.go",
        """package server

import (
    \"errors\"
    \"net/http\"
    \"time\"
)

func (s *Server) handleWeightToday(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    today := localDayString(time.Now())

    switch r.Method {
    case http.MethodGet:
        entry, err := s.db.GetWeightForDay(ctx, today)
        if err != nil {
            writeError(w, http.StatusInternalServerError, err)
            return
        }
        writeJSON(w, http.StatusOK, map[string]any{\"today\": today, \"entry\": entry})

    case http.MethodPut:
        var body struct {
            Value float64 `json:\"value\"`
            Unit  string  `json:\"unit\"`
        }
        if err := parseJSON(r, &body); err != nil {
            writeError(w, http.StatusBadRequest, err)
            return
        }
        if body.Value <= 0 {
            writeError(w, http.StatusBadRequest, errors.New(\"value must be > 0\"))
            return
        }
        if body.Unit != \"kg\" && body.Unit != \"lb\" {
            writeError(w, http.StatusBadRequest, errors.New(\"unit must be \\\"kg\\\" or \\\"lb\\\"\"))
            return
        }

        if err := s.db.UpsertWeightForDay(ctx, today, body.Value, body.Unit, time.Now()); err != nil {
            writeError(w, http.StatusInternalServerError, err)
            return
        }
        entry, _ := s.db.GetWeightForDay(ctx, today)
        writeJSON(w, http.StatusOK, map[string]any{\"today\": today, \"entry\": entry})

    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
}

func (s *Server) handleWeightRecent(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    limit := intQuery(r, \"limit\", 14)
    items, err := s.db.ListRecentWeights(r.Context(), limit)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{\"items\": items})
}
""",
    )

    write(
        "internal/server/handlers_water.go",
        """package server

import (
    \"errors\"
    \"net/http\"
    \"time\"
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
    writeJSON(w, http.StatusOK, map[string]any{\"today\": today, \"totalLiters\": total})
}

func (s *Server) handleWaterEvent(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    var body struct {
        DeltaLiters float64 `json:\"deltaLiters\"`
    }
    if err := parseJSON(r, &body); err != nil {
        writeError(w, http.StatusBadRequest, err)
        return
    }
    if body.DeltaLiters == 0 || body.DeltaLiters < -10 || body.DeltaLiters > 10 {
        writeError(w, http.StatusBadRequest, errors.New(\"deltaLiters must be non-zero and within [-10, 10]\"))
        return
    }
    id, err := s.db.AddWaterEvent(r.Context(), body.DeltaLiters, time.Now())
    if err != nil {
        writeError(w, http.StatusInternalServerError, err)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{\"id\": id})
}

func (s *Server) handleWaterRecent(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    limit := intQuery(r, \"limit\", 20)
    items, err := s.db.ListRecentWaterEvents(r.Context(), limit)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{\"items\": items})
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
        writeJSON(w, http.StatusOK, map[string]any{\"undone\": false})
        return
    }
    if err := s.db.DeleteWaterEvent(r.Context(), items[0].ID); err != nil {
        writeError(w, http.StatusInternalServerError, err)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{\"undone\": true, \"id\": items[0].ID})
}
""",
    )

    print("Rewrote Go files.")


if __name__ == "__main__":
    main()
