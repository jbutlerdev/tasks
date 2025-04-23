package api

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jbutlerdev/tasks/internal/storage"
)

// HTMXMiddleware adds support for HTMX headers and better error handling
func HTMXMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log incoming requests
		log.Printf("%s %s", r.Method, r.URL.Path)
		
		// Store whether this is an HTMX request for access in error handlers
		isHtmx := r.Header.Get("HX-Request") == "true"
		if isHtmx {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		
		// Create a custom response writer that can detect errors
		next.ServeHTTP(w, r)
	})
}

func NewRouter(store *storage.FileStore) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(HTMXMiddleware)

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Route("/lists", func(r chi.Router) {
			r.Get("/", HandleGetAllLists(store))
			r.Post("/", HandleCreateList(store))
			r.Route("/{listID}", func(r chi.Router) {
				r.Get("/", HandleGetList(store))
				r.Put("/", HandleUpdateList(store))
				r.Delete("/", HandleDeleteList(store))
				r.Get("/tasks", HandleGetTasksForList(store))
				r.Post("/tasks", HandleCreateTask(store))
			})
		})

		r.Route("/tasks", func(r chi.Router) {
			r.Get("/", HandleGetAllTasks(store))
			r.Route("/{listID}/{taskID}", func(r chi.Router) {
				r.Get("/", HandleGetTask(store))
				r.Put("/", HandleUpdateTask(store))
				r.Delete("/", HandleDeleteTask(store))
			})
		})

		// Export endpoint
		r.Get("/export", HandleExportMarkdown(store))
		
		// OpenAPI specification endpoint
		r.Get("/openapi", HandleOpenAPISpec(store))
	})

	// Web UI routes
	r.Route("/", func(r chi.Router) {
		// Serve static files
		fileServer := http.FileServer(http.Dir("./web/static"))
		r.Handle("/static/*", http.StripPrefix("/static", fileServer))

		// UI routes
		r.Get("/", HandleHomeUI(store))
		r.Get("/lists", HandleListsUI(store))
		r.Get("/lists/{listID}", HandleListUI(store))
		r.Get("/kanban/{listID}", HandleKanbanUI(store))
	})

	return r
}

