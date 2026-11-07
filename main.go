package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"edgenet/internal/portal"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("load .env: %v", err)
	}

	key := os.Getenv("ADMIN_TOKEN")
	if key != "" && len(key) < 16 {
		log.Fatal("set ADMIN_TOKEN to a secret of at least 16 characters")
	}
	dir := os.Getenv("DATA_DIR")
	if dir == "" {
		dir = "data"
	}
	s, err := portal.Open(dir, key)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "http://localhost:5173"
	}
	if err := s.ConfigureAuth(portal.AuthConfig{
		AppURL:             appURL,
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	}); err != nil {
		log.Fatal(err)
	}
	if key == "" && os.Getenv("GOOGLE_CLIENT_ID") == "" {
		log.Fatal("configure GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET, or ADMIN_TOKEN for local administration")
	}
	app := s.App()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() { <-stop; _ = app.Shutdown() }()
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	if err := app.Listen(addr); err != nil {
		log.Printf("Server stopped: %v", err)
	}
}
