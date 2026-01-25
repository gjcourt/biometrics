package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"biometrics/internal/db"
	"biometrics/internal/server"
)

func main() {
	addr := env("ADDR", ":8080")
	dbPath := env("DB_PATH", "biometrics.sqlite")
	webDir := env("WEB_DIR", "web")

	d, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer func() { _ = d.Close() }()

	h := server.New(d, webDir).Handler()
	log.Printf("listening on %s (db: %s)", addr, dbPath)
	if err := http.ListenAndServe(addr, h); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
