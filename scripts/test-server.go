package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	// Get the examples directory
	examplesDir := "./examples"
	if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
		log.Fatal("Examples directory not found")
	}

	// Create file server
	fs := http.FileServer(http.Dir(examplesDir))

	// Handle root to show available files
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			// Combine all HTML into a single write to avoid multiple error checks
			html := `<h1>Sitemap Test Server</h1>
<p>Available sitemaps:</p>
<ul>
<li><a href='/sample-sitemap.xml'>Sample Sitemap</a></li>
<li><a href='/sitemap-index.xml'>Sitemap Index</a></li>
<li><a href='/plain-sitemap.txt'>Plain Text Sitemap</a></li>
</ul>`
			if _, err := fmt.Fprint(w, html); err != nil {
				http.Error(w, "Failed to write response", http.StatusInternalServerError)
				return
			}
			return
		}
		fs.ServeHTTP(w, r)
	})

	port := ":8080"
	fmt.Printf("Starting test server on http://localhost%s\n", port)
	fmt.Printf("Available sitemaps:\n")
	fmt.Printf("  - http://localhost%s/sample-sitemap.xml\n", port)
	fmt.Printf("  - http://localhost%s/sitemap-index.xml\n", port)
	fmt.Printf("  - http://localhost%s/plain-sitemap.txt\n", port)
	fmt.Printf("\nPress Ctrl+C to stop\n")

	// Create server with proper timeouts to address G114
	server := &http.Server{
		Addr:         port,
		Handler:      nil, // Use DefaultServeMux
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}
