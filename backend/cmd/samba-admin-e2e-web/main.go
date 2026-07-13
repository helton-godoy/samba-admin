// Command samba-admin-e2e-web serve o bundle de teste e encaminha a API em
// loopback. Ele existe somente para E2E e não integra os binários do package.
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	root := strings.TrimSpace(os.Getenv("SAMBA_ADMIN_E2E_FRONTEND_DIST"))
	address := strings.TrimSpace(os.Getenv("SAMBA_ADMIN_E2E_FRONTEND_ADDR"))
	apiValue := strings.TrimSpace(os.Getenv("SAMBA_ADMIN_E2E_API_URL"))
	if address == "" {
		address = "127.0.0.1:4174"
	}
	if root == "" || !filepath.IsAbs(root) {
		log.Fatal("SAMBA_ADMIN_E2E_FRONTEND_DIST deve ser um diretório absoluto")
	}
	if address != "127.0.0.1:4174" && !strings.HasPrefix(address, "127.0.0.1:") {
		log.Fatal("o servidor E2E deve escutar exclusivamente em 127.0.0.1")
	}
	target, err := url.Parse(apiValue)
	if err != nil || target.Scheme != "http" || target.Hostname() != "127.0.0.1" || target.Port() == "" {
		log.Fatal("SAMBA_ADMIN_E2E_API_URL deve usar HTTP, loopback e porta explícita")
	}
	index := filepath.Join(root, "index.html")
	if info, statErr := os.Stat(index); statErr != nil || !info.Mode().IsRegular() {
		log.Fatal("index.html ausente no frontend-dist E2E")
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, proxyErr error) {
		http.Error(w, fmt.Sprintf("API E2E indisponível: %v", proxyErr), http.StatusBadGateway)
	}
	files := http.FileServer(http.Dir(root))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			proxy.ServeHTTP(w, r)
			return
		}
		clean := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if clean == "." {
			clean = "index.html"
		}
		candidate := filepath.Join(root, clean)
		if relative, relativeErr := filepath.Rel(root, candidate); relativeErr != nil || strings.HasPrefix(relative, "..") {
			http.Error(w, "caminho inválido", http.StatusBadRequest)
			return
		}
		if info, statErr := os.Stat(candidate); statErr == nil && info.Mode().IsRegular() {
			files.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, index)
	})

	server := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("frontend E2E disponível em http://%s", address)
	if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
