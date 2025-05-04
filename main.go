package main

import (
	"log/slog"
	"os"
	"ustawka/server"
)

func main() {
	// Configure slog
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create and start server
	srv, err := server.NewServer()
	if err != nil {
		slog.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	if err := srv.Start("8080"); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
