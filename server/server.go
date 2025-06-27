package server

import (
	"html/template"
	"log/slog"
	"net/http"
	"os"
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
	router  *chi.Mux
	handler *handlers.Handler
}

// NewServer creates a new server instance with all dependencies
func NewServer() (*Server, error) {
	// Load templates
	templates := template.Must(template.ParseFiles(
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

	return &Server{
		router:  r,
		handler: handler,
	}, nil
}

// Start starts the HTTP server on the specified port
func (s *Server) Start(port string) error {
	slog.Info("Server starting", "port", port)
	return http.ListenAndServe(":"+port, s.router)
}
