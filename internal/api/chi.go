package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/SamuelSchutz13/uJobsGUI/internal/core"
)

func Chi(s *core.Service, static http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	reply := func(w http.ResponseWriter, v any, err error) {
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(Status(err))
			v = Body(err)
		}
		json.NewEncoder(w).Encode(v)
	}

	r.Route("/api", func(r chi.Router) {
		r.Get("/info", func(w http.ResponseWriter, _ *http.Request) { reply(w, Info{s.Root, "chi"}, nil) })
		r.Get("/tracker", func(w http.ResponseWriter, _ *http.Request) { v, err := s.Tracker(); reply(w, v, err) })
		r.Get("/jobs", func(w http.ResponseWriter, _ *http.Request) { v, err := s.Jobs(); reply(w, v, err) })
		r.Post("/jobs/status", func(w http.ResponseWriter, req *http.Request) {
			var body JobStatusReq
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(BadRequest(err))
				return
			}
			reply(w, body, s.SetJobStatus(body.Key, body.Status))
		})
		r.Get("/portals", func(w http.ResponseWriter, _ *http.Request) { v, err := s.Portals(); reply(w, v, err) })
		r.Get("/search", func(w http.ResponseWriter, req *http.Request) {
			q := req.URL.Query()
			v, err := s.Search(req.Context(), q.Get("portal"), q.Get("q"))
			reply(w, v, err)
		})
		r.Get("/commands", func(w http.ResponseWriter, _ *http.Request) { v, err := s.Commands(); reply(w, v, err) })
		r.Get("/runs", func(w http.ResponseWriter, _ *http.Request) { reply(w, s.Runs(), nil) })
		r.Post("/runs", func(w http.ResponseWriter, req *http.Request) {
			var body StartReq
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(BadRequest(err))
				return
			}
			v, err := s.Start(body.Command, body.Args)
			reply(w, v, err)
		})
		r.Get("/runs/{id}", func(w http.ResponseWriter, req *http.Request) {
			v, err := s.Run(chi.URLParam(req, "id"))
			reply(w, v, err)
		})
		r.Post("/runs/{id}/reply", func(w http.ResponseWriter, req *http.Request) {
			var body ReplyReq
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(BadRequest(err))
				return
			}
			v, err := s.Reply(chi.URLParam(req, "id"), body.Message)
			reply(w, v, err)
		})
	})
	r.Handle("/files/*", s.Files())
	r.Handle("/*", static)
	return r
}
