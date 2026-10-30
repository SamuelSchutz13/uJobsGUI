package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/SamuelSchutz13/uJobsGUI/internal/core"
)

// fakeCheckout builds the minimum ai-job-search layout the GUI reads.
func fakeCheckout(t *testing.T) string {
	root := t.TempDir()
	write := func(name, body string) {
		p := filepath.Join(root, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".claude/commands/apply.md", "apply")
	write(".claude/skills/job-scraper/SKILL.md", "---\nname: scrape\n---\n")
	write(".agents/skills/jobindex-search/cli/src/cli.ts", "")
	write("job_search_tracker.csv", "\xef\xbb\xbfdate,company,role,status,cv_file\n2026-10-01,Acme,Engineer,drafted,cv/main_acme.pdf\n2026-10-02,Old,Dev\n")
	write("job_scraper/seen_jobs.json", `{"version":2,"seen":{"k1":{"title":"Dev","company":"Acme","status":"ranked","rank_score":81}}}`)
	write("cv/main_acme.pdf", "%PDF")
	write("profile.md", "secret")
	return root
}

// Every router must answer the same requests identically.
func TestRoutersAgree(t *testing.T) {
	s, err := core.New(fakeCheckout(t), "claude-missing", "bun-missing")
	if err != nil {
		t.Fatal(err)
	}
	web := fstest.MapFS{"index.html": {Data: []byte("<html>spa</html>")}, "app.js": {Data: []byte("js")}}

	cases := []struct {
		method, path, ctype, body string
		code                      int
		contains                  string
	}{
		{"GET", "/api/info", "", "", 200, `"root":`},
		{"GET", "/api/tracker", "", "", 200, `"company":"Acme"`},
		{"GET", "/api/tracker", "", "", 200, `"company":"Old"`},
		{"GET", "/api/jobs", "", "", 200, `"key":"k1"`},
		{"GET", "/api/portals", "", "", 200, `["jobindex-search"]`},
		{"POST", "/api/jobs/status", "application/json", `{"key":"k1","status":"skipped"}`, 200, `"status":"skipped"`},
		{"GET", "/api/jobs", "", "", 200, `"status":"skipped"`},
		{"POST", "/api/jobs/status", "application/json", `{"key":"k1","status":"ranked"}`, 200, `"status":"ranked"`},
		{"POST", "/api/jobs/status", "application/json", `{"key":"k1","status":"deleted"}`, 400, "inválida"},
		{"POST", "/api/jobs/status", "application/json", `{"key":"nope","status":"skipped"}`, 404, "error"},
		{"GET", "/api/commands", "", "", 200, `["apply","scrape"]`},
		{"GET", "/api/search?portal=nope&q=x", "", "", 404, "error"},
		{"GET", "/api/search?portal=jobindex-search&q=", "", "", 400, "error"},
		{"GET", "/api/runs/none", "", "", 404, "error"},
		{"POST", "/api/runs", "application/json", `{"command":"rm -rf"}`, 400, "comando desconhecido"},
		{"POST", "/api/runs", "text/plain", `{"command":"apply"}`, 415, ""},
		{"GET", "/files/cv/main_acme.pdf", "", "", 200, "%PDF"},
		{"GET", "/files/profile.md", "", "", 404, ""},
		{"GET", "/files/cv/../profile.md", "", "", 404, ""},
		{"GET", "/app.js", "", "", 200, "js"},
		{"GET", "/candidaturas", "", "", 200, "spa"},
	}
	for _, router := range Routers {
		h, err := New(router, s, web)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range cases {
			req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
			req.Host = "localhost:8080"
			if c.ctype != "" {
				req.Header.Set("Content-Type", c.ctype)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != c.code || !strings.Contains(rec.Body.String(), c.contains) {
				t.Errorf("%s %s %s: got %d %q, want %d containing %q", router, c.method, c.path, rec.Code, rec.Body.String(), c.code, c.contains)
			}
		}

		// A foreign Host (DNS rebinding) is refused.
		req := httptest.NewRequest("GET", "/api/tracker", nil)
		req.Host = "evil.example:8080"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: foreign host got %d", router, rec.Code)
		}
	}
}

// Removing a posting must keep the rest of the file the scraper owns.
func TestSetJobStatusKeepsFields(t *testing.T) {
	root := fakeCheckout(t)
	s, _ := core.New(root, "", "")
	if err := s.SetJobStatus("k1", "skipped"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "job_scraper", "seen_jobs.json"))
	var doc struct {
		Version int                       `json:"version"`
		Seen    map[string]map[string]any `json:"seen"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	e := doc.Seen["k1"]
	if doc.Version != 2 || e["status"] != "skipped" || e["rank_score"] != 81.0 || e["company"] != "Acme" {
		t.Fatalf("file changed beyond the status: %s", data)
	}
}

func TestStartRecordsFailedRun(t *testing.T) {
	s, err := core.New(fakeCheckout(t), "claude-missing", "bun-missing")
	if err != nil {
		t.Fatal(err)
	}
	h, _ := New("chi", s, fstest.MapFS{})
	req := httptest.NewRequest("POST", "/api/runs", strings.NewReader(`{"command":"apply","args":"https://x"}`))
	req.Host = "127.0.0.1:8080"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var run core.Run
	if err := json.Unmarshal(rec.Body.Bytes(), &run); err != nil || run.Turns[0].Text != "/apply https://x" {
		t.Fatalf("start: %d %s", rec.Code, rec.Body)
	}
	for i := 0; i < 200; i++ { // the missing executable fails fast
		if r, _ := s.Run(run.ID); !r.Running {
			if len(r.Turns) != 2 || !strings.Contains(r.Turns[1].Text, "[erro]") {
				t.Fatalf("want error turn, got %+v", r.Turns)
			}
			return
		}
		<-time.After(10 * time.Millisecond)
	}
	t.Fatal("run never finished")
}
