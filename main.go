package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/jbutlerdev/tasks/internal/api"
	"github.com/jbutlerdev/tasks/internal/storage"
)

func main() {
	port := flag.Int("port", 8080, "Port to run the server on")
	dataDir := flag.String("data", "./data", "Directory to store task data")
	flag.Parse()

	// Initialize storage
	store, err := storage.NewFileStore(*dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Setup API routes
	router := api.NewRouter(store)

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Server starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}
