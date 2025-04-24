package api

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"

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

func NewRouter(store *storage.FileStore, staticFS embed.FS) http.Handler {
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
		// Create a custom file server to set proper content types
		staticSubFS, err := fs.Sub(staticFS, "web/static")
		if err != nil {
			log.Fatalf("Failed to create sub-filesystem for static files: %v", err)
		}
		
		// Custom file server that ensures correct MIME types
		fileServer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			
			// Set correct content types based on file extension
			if strings.HasSuffix(path, ".css") {
				w.Header().Set("Content-Type", "text/css")
			} else if strings.HasSuffix(path, ".js") {
				w.Header().Set("Content-Type", "application/javascript")
			} else if strings.HasSuffix(path, ".ico") {
				w.Header().Set("Content-Type", "image/x-icon")
			}
			
			// Pass to the standard file server
			http.FileServer(http.FS(staticSubFS)).ServeHTTP(w, r)
		})
		
		r.Handle("/static/*", http.StripPrefix("/static", fileServer))

		// UI routes
		r.Get("/", HandleHomeUI(store))
		r.Get("/lists", HandleListsUI(store))
		r.Get("/lists/{listID}", HandleListUI(store))
		r.Get("/kanban/{listID}", HandleKanbanUI(store))
		r.Get("/all-kanban", HandleAllKanbanUI(store))
	})

	return r
}