package server

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"ustawka/db"
	"ustawka/handlers"
	"ustawka/sejm"
	"ustawka/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Server represents the HTTP server instance
type Server struct {
	router            *chi.Mux
	handler           *handlers.Handler
	backgroundService *service.BackgroundService
}

// NewServer creates a new server instance with all dependencies
func NewServer() (*Server, error) {
	// Load templates with custom functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
	}
	
	templates := template.Must(template.New("").Funcs(funcMap).ParseFiles(
		"templates/base.html",
		"templates/board.html",
		"templates/act_details.html",
	))

	// Create SEJM client
	sejmClient := sejm.NewClient()

	// Initialize database
	dbPath := os.Getenv("SEJM_DB_PATH")
	if dbPath == "" {
		dbPath = "sejm.db"
	}
	database, err := db.New(dbPath)
	if err != nil {
		return nil, err
	}

	// Create service layer with the concrete client and database
	actService := service.NewActService(sejmClient, database)

	// Create Senate client for enhanced features
	senateClient := sejm.NewSimpleSenateClient()

	// Create enrichment service
	enrichmentService := service.NewEnrichmentService(sejmClient, senateClient)

	// Create data pipeline
	pipelineConfig := service.DefaultPipelineConfig()
	pipeline := service.NewPipeline(sejmClient, senateClient, database, pipelineConfig)

	// Create background service
	backgroundConfig := service.DefaultBackgroundConfig()
	backgroundService := service.NewBackgroundService(
		pipeline, enrichmentService, database, sejmClient, backgroundConfig)

	// Create handler
	handler := handlers.NewHandler(templates, actService)

	// Create router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Serve static files
	fileServer := http.FileServer(http.Dir("static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Routes
	r.Get("/", handler.Home)
	r.Get("/api/years", handler.HandleYears)
	r.Get("/api/acts/DU/{year}", handler.HandleActs)
	r.Get("/api/acts/DU/{year}/{position}", handler.HandleActDetails)
	r.Get("/acts/DU/{year}/{position}", handler.ViewActDetails)
	r.Get("/metrics", handlers.MetricsHandler)

	// Background service management routes
	r.Get("/api/background/status", func(w http.ResponseWriter, _ *http.Request) {
		status := backgroundService.GetStatus()
		w.Header().Set("Content-Type", "application/json")
		if err := handlers.WriteJSON(w, status); err != nil {
			http.Error(w, "Failed to encode status", http.StatusInternalServerError)
		}
	})

	r.Post("/api/background/sync", func(w http.ResponseWriter, r *http.Request) {
		if err := backgroundService.TriggerSync(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"message": "Sync triggered successfully"}`))
	})

	r.Post("/api/background/enrich", func(w http.ResponseWriter, r *http.Request) {
		if err := backgroundService.TriggerEnrichment(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"message": "Enrichment triggered successfully"}`))
	})

	return &Server{
		router:            r,
		handler:           handler,
		backgroundService: backgroundService,
	}, nil
}

// Start starts the HTTP server and background services on the specified port
func (s *Server) Start(port string) error {
	ctx := context.Background()
	
	// Start background service
	if err := s.backgroundService.Start(ctx); err != nil {
		slog.Error("Failed to start background service", "error", err)
		return err
	}
	
	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	server := &http.Server{
		Addr:    ":" + port,
		Handler: s.router,
	}
	
	// Start server in a goroutine
	go func() {
		slog.Info("HTTP server starting", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server failed", "error", err)
		}
	}()
	
	// Wait for interrupt signal
	<-c
	slog.Info("Shutting down server...")
	
	// Stop background service
	s.backgroundService.Stop()
	
	// Shutdown HTTP server with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		return err
	}
	
	slog.Info("Server exited")
	return nil
}
