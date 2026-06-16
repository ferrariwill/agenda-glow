// Servidor estático de desenvolvimento para preview dos templates HTML (porta 8082).
package main

import (
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	port := strings.TrimSpace(os.Getenv("FRONTEND_PORT"))
	if port == "" {
		port = "8082"
	}

	root := strings.TrimSpace(os.Getenv("FRONTEND_ROOT"))
	if root == "" {
		root = "frontend/templates"
	}

	addr := ":" + port
	fs := http.FileServer(http.Dir(root))

	mux := http.NewServeMux()
	mux.Handle("/", fs)

	log.Printf("AgendaGlow front-end dev ouvindo em http://localhost%s (dir: %s)", addr, root)
	log.Printf("Abra ex.: http://localhost%s/dashboard_dona.html", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("servidor encerrado: %v", err)
	}
}
