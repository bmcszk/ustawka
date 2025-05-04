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

	// Get port from environment variable or use default
	port := os.Getenv("USTAWKA_PORT")
	if port == "" {
		port = "8080"
		slog.Info("Using default port", "port", port)
	} else {
		slog.Info("Using custom port", "port", port)
	}

	// Create and start server
	srv, err := server.NewServer()
	if err != nil {
		slog.Error("Failed to create server", "error", err)
		panic(err)
	}

	if err := srv.Start(port); err != nil {
		slog.Error("Server failed to start", "error", err)
		panic(err)
	}
}
