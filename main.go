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
	if len(key) < 16 {
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
