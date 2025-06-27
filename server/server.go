package server

import (
	"context"
	"errors"
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
	templates, err := loadTemplates()
	if err != nil {
		return nil, err
	}

	database, err := initializeDatabase()
	if err != nil {
		return nil, err
	}

	services, err := createServices(database)
	if err != nil {
		return nil, err
	}

	handler := handlers.NewHandler(templates, services.ActService, services.SearchService, services.ComparisonService)
	router := createRouter(handler, services.BackgroundService)

	return &Server{
		router:            router,
		handler:           handler,
		backgroundService: services.BackgroundService,
	}, nil
}

// Services holds all application services
type Services struct {
	ActService        *service.ActService
	SearchService     *service.SearchService
	ComparisonService *service.ComparisonService
	BackgroundService *service.BackgroundService
}

func loadTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
	}
	
	return template.New("").Funcs(funcMap).ParseFiles(
		"templates/base.html",
		"templates/board.html",
		"templates/act_details.html",
		"templates/search_results.html",
		"templates/comparison_results.html",
	)
}

func initializeDatabase() (service.Database, error) {
	dbPath := os.Getenv("SEJM_DB_PATH")
	if dbPath == "" {
		dbPath = "sejm.db"
	}
	return db.New(dbPath)
}

func createServices(database service.Database) (*Services, error) {
	sejmClient := sejm.NewClient()
	senateClient := sejm.NewSimpleSenateClient()

	actService := service.NewActService(sejmClient, database)
	searchService := service.NewSearchService(database)
	comparisonService := service.NewComparisonService(database)

	enrichmentService := service.NewEnrichmentService(sejmClient, senateClient)
	pipelineConfig := service.DefaultPipelineConfig()
	
	// Type assertion for pipeline which needs concrete DB type
	concreteDB, ok := database.(*db.DB)
	if !ok {
		return nil, errors.New("database must be *db.DB type for pipeline")
	}
	pipeline := service.NewPipeline(sejmClient, senateClient, concreteDB, pipelineConfig)

	backgroundConfig := service.DefaultBackgroundConfig()
	backgroundService := service.NewBackgroundService(
		pipeline, enrichmentService, database, sejmClient, backgroundConfig)

	return &Services{
		ActService:        actService,
		SearchService:     searchService,
		ComparisonService: comparisonService,
		BackgroundService: backgroundService,
	}, nil
}

func createRouter(handler *handlers.Handler, backgroundService *service.BackgroundService) *chi.Mux {
	r := chi.NewRouter()
	setupMiddleware(r)
	setupStaticFiles(r)
	setupRoutes(r, handler)
	setupBackgroundRoutes(r, backgroundService)
	return r
}

func setupMiddleware(r *chi.Mux) {
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
}

func setupStaticFiles(r *chi.Mux) {
	fileServer := http.FileServer(http.Dir("static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))
}

func setupRoutes(r *chi.Mux, handler *handlers.Handler) {
	r.Get("/", handler.Home)
	r.Get("/api/years", handler.HandleYears)
	r.Get("/api/acts/DU/{year}", handler.HandleActs)
	r.Get("/api/acts/DU/{year}/{position}", handler.HandleActDetails)
	r.Get("/acts/DU/{year}/{position}", handler.ViewActDetails)
	r.Get("/metrics", handlers.MetricsHandler)

	r.Get("/api/search", handler.HandleSearch)
	r.Get("/api/search/suggestions", handler.HandleSearchSuggestions)
	r.Get("/api/search/facets", handler.HandleSearchFacets)

	r.Get("/api/compare", handler.HandleCompareActs)
	r.Get("/api/compare/suggestions", handler.HandleComparisonSuggestions)
}

func setupBackgroundRoutes(r *chi.Mux, backgroundService *service.BackgroundService) {
	r.Get("/api/background/status", createStatusHandler(backgroundService))
	r.Post("/api/background/sync", createSyncHandler(backgroundService))
	r.Post("/api/background/enrich", createEnrichHandler(backgroundService))
	r.Get("/api/monitoring/stats", createMonitoringHandler(backgroundService))
	r.Get("/api/validation/stats", createValidationHandler(backgroundService))
}

func createStatusHandler(backgroundService *service.BackgroundService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		status := backgroundService.GetStatus()
		w.Header().Set("Content-Type", "application/json")
		if err := handlers.WriteJSON(w, status); err != nil {
			http.Error(w, "Failed to encode status", http.StatusInternalServerError)
		}
	}
}

func createSyncHandler(backgroundService *service.BackgroundService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := backgroundService.TriggerSync(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"message": "Sync triggered successfully"}`))
	}
}

func createEnrichHandler(backgroundService *service.BackgroundService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := backgroundService.TriggerEnrichment(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"message": "Enrichment triggered successfully"}`))
	}
}

func createMonitoringHandler(backgroundService *service.BackgroundService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		stats := backgroundService.GetMonitoringStats()
		if err := handlers.WriteJSON(w, stats); err != nil {
			http.Error(w, "Failed to encode monitoring stats", http.StatusInternalServerError)
		}
	}
}

func createValidationHandler(backgroundService *service.BackgroundService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		stats := backgroundService.GetValidationStats()
		if err := handlers.WriteJSON(w, stats); err != nil {
			http.Error(w, "Failed to encode validation stats", http.StatusInternalServerError)
		}
	}
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
