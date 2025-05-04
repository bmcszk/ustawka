package server

import (
	"html/template"
	"log/slog"
	"net/http"
	"ustawka/handlers"
	"ustawka/sejm"
	"ustawka/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	router  *chi.Mux
	handler *handlers.Handler
}

func NewServer() (*Server, error) {
	// Load templates
	templates := template.Must(template.ParseFiles("templates/base.html", "templates/kanban.html"))

	// Create SEJM client
	sejmClient := sejm.NewClient()

	// Create service layer with the concrete client
	actService := service.NewActService(sejmClient)

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
	r.Get("/api/years", handler.GetYears)
	r.Get("/api/acts/DU/{year}", handler.GetActs)
	r.Get("/api/acts/DU/{year}/{position}", handler.GetActDetails)

	return &Server{
		router:  r,
		handler: handler,
	}, nil
}

func (s *Server) Start(port string) error {
	slog.Info("Server starting", "port", port)
	return http.ListenAndServe(":"+port, s.router)
}
