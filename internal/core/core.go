// Package core wraps an ai-job-search checkout. It never reimplements the
// workflow: it reads the files the workflow writes and runs its own CLIs.
package core

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
	ErrBadInput = errors.New("bad input")
)

// Error is a failure the user sees. Code lets the frontend say it in the viewer's
// language; Msg is the English fallback for logs and other clients.
type Error struct {
	Kind error  // ErrBadInput or ErrNotFound
	Code string // e.g. "unknown_command"
	Arg  string // the value the message names, if any
	Msg  string
}

func (e *Error) Error() string { return e.Msg }
func (e *Error) Unwrap() error { return e.Kind }

func bad(code, arg, msg string) error      { return &Error{ErrBadInput, code, arg, msg} }
func notFound(code, arg, msg string) error { return &Error{ErrNotFound, code, arg, msg} }

// Service is the framework-agnostic API every router adapter calls.
type Service struct {
	Root   string // ai-job-search checkout
	Claude string // claude executable
	Bun    string // bun executable

	mu   sync.Mutex
	runs map[string]*Run
}

func New(root, claude, bun string) (*Service, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "commands", "apply.md")); err != nil {
		return nil, fmt.Errorf("%s does not look like an ai-job-search folder (no .claude/commands/apply.md)", root)
	}
	return &Service{Root: root, Claude: claude, Bun: bun, runs: map[string]*Run{}}, nil
}

// Tracker returns job_search_tracker.csv as one map per row; no file means no applications yet.
func (s *Service) Tracker() ([]map[string]string, error) {
	f, err := os.Open(filepath.Join(s.Root, "job_search_tracker.csv"))
	if errors.Is(err, fs.ErrNotExist) {
		return []map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(bufio.NewReader(f))
	r.FieldsPerRecord = -1 // legacy rows lack the later `deadline` column
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("job_search_tracker.csv: %w", err)
	}
	out := []map[string]string{}
	if len(rows) == 0 {
		return out, nil
	}
	head := rows[0]
	head[0] = strings.TrimPrefix(head[0], "\xef\xbb\xbf") // spreadsheet "CSV UTF-8" adds a BOM
	for _, row := range rows[1:] {
		m := map[string]string{}
		for i, h := range head {
			if i < len(row) {
				m[h] = row[i]
			}
		}
		out = append(out, m)
	}
	return out, nil
}

// Jobs returns the /scrape + /rank history (job_scraper/seen_jobs.json) as a list, key included.
func (s *Service) Jobs() ([]map[string]any, error) {
	data, err := os.ReadFile(filepath.Join(s.Root, "job_scraper", "seen_jobs.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return []map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("seen_jobs.json: %w", err)
	}
	// Same rule as tools/rank_state.py load_state: entries live under "seen" or at the top level.
	if inner, ok := doc["seen"]; ok {
		doc = nil
		if err := json.Unmarshal(inner, &doc); err != nil {
			return nil, fmt.Errorf("seen_jobs.json: %w", err)
		}
	}
	out := []map[string]any{}
	for key, raw := range doc {
		var e map[string]any
		if json.Unmarshal(raw, &e) != nil {
			continue
		}
		e["key"] = key
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return fmt.Sprint(out[i]["key"]) < fmt.Sprint(out[j]["key"]) })
	return out, nil
}

// jobStatuses is the seen_jobs.json status vocabulary of the job-scraper skill.
var jobStatuses = []string{"new", "skipped", "ranked", "expired", "unreachable"}

// SetJobStatus changes one entry's status and keeps every other field. The GUI's
// "remove" is status "skipped": /rank never selects it and /scrape still dedups against it.
func (s *Service) SetJobStatus(key, status string) error {
	if !slices.Contains(jobStatuses, status) {
		return bad("bad_status", status, fmt.Sprintf("invalid status %q", status))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.Root, "job_scraper", "seen_jobs.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return notFound("no_jobs_file", "", "job_scraper/seen_jobs.json does not exist yet")
	}
	if err != nil {
		return err
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("seen_jobs.json: %w", err)
	}
	seen := doc
	if inner, ok := doc["seen"]; ok {
		seen = nil // a fresh map: decoding into doc itself would merge the two levels
		if err := json.Unmarshal(inner, &seen); err != nil {
			return fmt.Errorf("seen_jobs.json: %w", err)
		}
	}
	var entry map[string]json.RawMessage
	if raw, ok := seen[key]; !ok || json.Unmarshal(raw, &entry) != nil {
		return notFound("job_not_found", key, fmt.Sprintf("job %q not found", key))
	}
	entry["status"], _ = json.Marshal(status)
	if seen[key], err = marshal(entry); err != nil {
		return err
	}
	if _, ok := doc["seen"]; ok {
		if doc["seen"], err = marshal(seen); err != nil {
			return err
		}
	}
	out, err := marshal(doc)
	if err != nil {
		return err
	}
	// Atomic replace, as rank_state.py save_state does: a half-written file loses the scrape history.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".seen_jobs.*.tmp")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(append(out, '\n'))
	cerr := tmp.Close()
	if werr == nil && cerr == nil {
		werr = os.Rename(tmp.Name(), path)
	}
	if werr != nil || cerr != nil {
		os.Remove(tmp.Name())
		return errors.Join(werr, cerr)
	}
	return nil
}

// marshal matches json.dump(indent=2, ensure_ascii=False) closely enough for a file other tools also write.
func marshal(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

// Portals lists the installed portal CLIs (.agents/skills/<name>-search/cli), discovered like install_portal_clis.py does.
func (s *Service) Portals() ([]string, error) {
	dirs, err := filepath.Glob(filepath.Join(s.Root, ".agents", "skills", "*", "cli", "src", "cli.ts"))
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, d := range dirs {
		out = append(out, filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(d)))))
	}
	return out, nil
}

// Search runs `bun run src/cli.ts search --query <q> --format json` for one portal and returns its JSON untouched.
func (s *Service) Search(ctx context.Context, portal, query string) (json.RawMessage, error) {
	portals, err := s.Portals()
	if err != nil {
		return nil, err
	}
	if !slices.Contains(portals, portal) {
		return nil, notFound("portal_not_found", portal, fmt.Sprintf("job board %q not found", portal))
	}
	if strings.TrimSpace(query) == "" {
		return nil, bad("empty_query", "", "type something to search for")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.Bun, "run", "src/cli.ts", "search", "--query", query, "--format", "json")
	cmd.Dir = filepath.Join(s.Root, ".agents", "skills", portal, "cli")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		// The CLIs write {"error","code"} to stderr; pass it on rather than a bare exit code.
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, bad("portal_failed", msg, portal+" failed: "+msg)
		}
		return nil, fmt.Errorf("%s: %w", portal, err)
	}
	if !json.Valid(stdout.Bytes()) {
		return nil, fmt.Errorf("%s returned invalid JSON", portal)
	}
	return stdout.Bytes(), nil
}

// Commands lists the slash commands the checkout ships: .claude/commands/*.md plus skills' frontmatter names.
func (s *Service) Commands() ([]string, error) {
	out := []string{}
	mds, _ := filepath.Glob(filepath.Join(s.Root, ".claude", "commands", "*.md"))
	for _, m := range mds {
		out = append(out, strings.TrimSuffix(filepath.Base(m), ".md"))
	}
	skills, _ := filepath.Glob(filepath.Join(s.Root, ".claude", "skills", "*", "SKILL.md"))
	for _, sk := range skills {
		data, err := os.ReadFile(sk)
		if err != nil {
			continue
		}
		for _, line := range strings.SplitN(string(data), "\n", 6) {
			if name, ok := strings.CutPrefix(strings.TrimSpace(line), "name:"); ok {
				out = append(out, strings.TrimSpace(name))
				break
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

// Turn is one message in a run's conversation.
type Turn struct {
	Role  string `json:"role"` // "user" or "claude"
	Text  string `json:"text"`
	Error bool   `json:"error,omitempty"` // claude exited with an error; Text holds its output and stderr
}

// Run is one Claude Code session started from the GUI; replies resume the same session.
type Run struct {
	ID      string    `json:"id"`
	Command string    `json:"command"`
	Started time.Time `json:"started"`
	Running bool      `json:"running"`
	Turns   []Turn    `json:"turns"`
}

// Start launches `claude -p "/<command> <args>"` in the checkout.
func (s *Service) Start(command, args string) (Run, error) {
	cmds, err := s.Commands()
	if err != nil {
		return Run{}, err
	}
	if !slices.Contains(cmds, command) {
		return Run{}, bad("unknown_command", command, fmt.Sprintf("unknown command %q", command))
	}
	r := &Run{ID: uuid(), Command: command, Started: time.Now()}
	s.mu.Lock()
	s.runs[r.ID] = r
	s.mu.Unlock()
	return s.turn(r, strings.TrimSpace("/"+command+" "+args), "--session-id", r.ID)
}

// Reply answers a question Claude asked, in the same session.
func (s *Service) Reply(id, message string) (Run, error) {
	if strings.TrimSpace(message) == "" {
		return Run{}, bad("empty_message", "", "the message is empty")
	}
	s.mu.Lock()
	r, ok := s.runs[id]
	s.mu.Unlock()
	if !ok {
		return Run{}, notFound("run_not_found", id, "run not found")
	}
	return s.turn(r, message, "--resume", id)
}

func (s *Service) turn(r *Run, prompt string, session ...string) (Run, error) {
	s.mu.Lock()
	if r.Running {
		s.mu.Unlock()
		return Run{}, bad("busy", "", "Claude is still answering")
	}
	r.Running = true
	r.Turns = append(r.Turns, Turn{Role: "user", Text: prompt})
	snap := snapshot(r)
	s.mu.Unlock()

	// acceptEdits lets the workflow write its CV/letter files; Bash stays limited to the
	// checkout's own .claude/settings.json allowlist, so its guards still apply.
	args := append([]string{"-p", prompt, "--permission-mode", "acceptEdits"}, session...)
	go func() {
		cmd := exec.Command(s.Claude, args...)
		cmd.Dir = s.Root
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		// shortcut: whole reply arrives at the end; switch to --output-format stream-json if live progress is wanted.
		out, err := cmd.Output()
		text := strings.TrimSpace(string(out))
		if err != nil {
			text = strings.TrimSpace(text + "\n\n" + err.Error() + "\n" + stderr.String())
		}
		s.mu.Lock()
		r.Turns = append(r.Turns, Turn{Role: "claude", Text: text, Error: err != nil})
		r.Running = false
		s.mu.Unlock()
	}()
	return snap, nil
}

func (s *Service) Run(id string) (Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.runs[id]
	if !ok {
		return Run{}, notFound("run_not_found", id, "run not found")
	}
	return snapshot(r), nil
}

func (s *Service) Runs() []Run {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Run{}
	for _, r := range s.runs {
		out = append(out, snapshot(r))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Started.After(out[j].Started) })
	return out
}

// Files serves the generated documents (/files/cv/..., /files/cover_letters/..., /files/documents/applications/...) and nothing else.
func (s *Service) Files() http.Handler {
	fsys := http.FileServerFS(os.DirFS(s.Root))
	return http.StripPrefix("/files", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := path.Clean(r.URL.Path)
		for _, dir := range []string{"/cv/", "/cover_letters/", "/documents/applications/"} {
			if strings.HasPrefix(p, dir) {
				fsys.ServeHTTP(w, r)
				return
			}
		}
		http.NotFound(w, r)
	}))
}

func snapshot(r *Run) Run {
	c := *r
	c.Turns = append([]Turn(nil), r.Turns...)
	return c
}

// uuid makes a v4 UUID, the form `claude --session-id` requires.
func uuid() string {
	var b [16]byte
	rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
