package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"wingmodels/internal/api"
	"wingmodels/internal/compiler"
	"wingmodels/internal/ui"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "compile":
		runCompile()
	case "serve":
		runServe()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: wingmodels <command>\n\nCommands:\n  compile    Run the compile pipeline\n  serve      Start the HTTP server\n")
}

func runCompile() {
	if err := compiler.Compile("data", "build"); err != nil {
		fmt.Fprintf(os.Stderr, "compile error: %v\n", err)
		os.Exit(1)
	}
}

func runServe() {
	snapshotPath := "build/api.json"

	apiServer, err := api.NewServer(snapshotPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "api server init error: %v\n", err)
		os.Exit(1)
	}

	uiServer, err := ui.NewServer(snapshotPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ui server init error: %v\n", err)
		os.Exit(1)
	}

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5, "text/html", "application/json", "text/css"))
	r.Use(middleware.Timeout(8 * time.Second))

	// Static assets
	r.Get("/output.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "output.css")
	})
	r.Handle("/public/*", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))

	// API routes
	r.Mount("/", apiServer.Router())

	// UI routes (registered directly to avoid chi Mount conflict on "/")
	uiServer.RegisterRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	addr := ":" + port
	fmt.Printf("Listening on %s\n", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
