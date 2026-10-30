// Package api exposes core.Service over HTTP. gin.go, echo.go and chi.go wire the
// same routes; everything framework-independent lives here.
//
//	GET  /api/info               checkout folder and router in use
//	GET  /api/tracker            job_search_tracker.csv
//	GET  /api/jobs               job_scraper/seen_jobs.json
//	GET  /api/portals            installed portal CLIs
//	GET  /api/search?portal=&q=  run one portal CLI
//	GET  /api/commands           slash commands of the checkout
//	GET  /api/runs               Claude Code runs started here
//	POST /api/runs               {"command","args"}
//	GET  /api/runs/{id}
//	POST /api/runs/{id}/reply    {"message"}
//	GET  /files/...              generated CVs, cover letters, archives
//	GET  /*                      Quasar SPA
package api

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"

	"github.com/SamuelSchutz13/uJobsGUI/internal/core"
)

var Routers = []string{"gin", "echo", "chi"}

// New builds the handler for the chosen router.
func New(router string, s *core.Service, web fs.FS) (http.Handler, error) {
	static := Static(web)
	var h http.Handler
	switch router {
	case "gin":
		h = Gin(s, static)
	case "echo":
		h = Echo(s, static)
	case "chi":
		h = Chi(s, static)
	default:
		return nil, fmt.Errorf("router %q desconhecido (use gin, echo ou chi)", router)
	}
	return Guard(h), nil
}

type Info struct {
	Root   string `json:"root"`
	Router string `json:"router"`
}

type StartReq struct {
	Command string `json:"command"`
	Args    string `json:"args"`
}

type ReplyReq struct {
	Message string `json:"message"`
}

type ErrBody struct {
	Error string `json:"error"`
}

// Status maps core errors to HTTP codes, identically for every router.
func Status(err error) int {
	switch {
	case errors.Is(err, core.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, core.ErrBadInput):
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

// Static serves the built SPA, falling back to index.html for client-side routes.
func Static(web fs.FS) http.Handler {
	files := http.FileServerFS(web)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if _, err := fs.Stat(web, name); name != "" && err == nil {
			files.ServeHTTP(w, r)
			return
		}
		if _, err := fs.Stat(web, "index.html"); err != nil {
			http.Error(w, "frontend não compilado: rode `npm run build` em web/", http.StatusNotFound)
			return
		}
		http.ServeFileFS(w, r, web, "index.html")
	})
}

// Guard blocks other websites from driving this local server: the Host must be
// loopback (stops DNS rebinding) and writes must be JSON (forces a CORS preflight we never grant).
func Guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if ip := net.ParseIP(strings.Trim(host, "[]")); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			http.Error(w, "host não permitido", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead &&
			!strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "use application/json", http.StatusUnsupportedMediaType)
			return
		}
		next.ServeHTTP(w, r)
	})
}
