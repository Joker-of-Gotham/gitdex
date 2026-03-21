package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/cli/command"
	ghclient "github.com/your-org/gitdex/internal/platform/github"
)

type fakeRelease struct {
	ID          int64      `json:"id"`
	TagName     string     `json:"tag_name"`
	Name        string     `json:"name,omitempty"`
	Body        string     `json:"body,omitempty"`
	Draft       bool       `json:"draft"`
	Prerelease  bool       `json:"prerelease"`
	HTMLURL     string     `json:"html_url,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

type releaseTestState struct {
	mu       sync.Mutex
	nextID   int64
	releases map[string]*fakeRelease
}

func newReleaseTestState() *releaseTestState {
	published := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	return &releaseTestState{
		nextID: 2,
		releases: map[string]*fakeRelease{
			"v1.0.0": {
				ID:          1,
				TagName:     "v1.0.0",
				Name:        "v1.0.0",
				HTMLURL:     "https://example.test/releases/v1.0.0",
				PublishedAt: &published,
			},
		},
	}
}

func (s *releaseTestState) list() []*fakeRelease {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*fakeRelease, 0, len(s.releases))
	for _, rel := range s.releases {
		clone := *rel
		out = append(out, &clone)
	}
	return out
}

func (s *releaseTestState) getByTag(tag string) (*fakeRelease, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rel, ok := s.releases[tag]
	if !ok {
		return nil, false
	}
	clone := *rel
	return &clone, true
}

func (s *releaseTestState) create(tag, name, body string, draft, prerelease bool) *fakeRelease {
	s.mu.Lock()
	defer s.mu.Unlock()
	var publishedAt *time.Time
	if !draft {
		now := time.Now().UTC()
		publishedAt = &now
	}
	rel := &fakeRelease{
		ID:          s.nextID,
		TagName:     tag,
		Name:        name,
		Body:        body,
		Draft:       draft,
		Prerelease:  prerelease,
		HTMLURL:     "https://example.test/releases/" + tag,
		PublishedAt: publishedAt,
	}
	s.nextID++
	s.releases[tag] = rel
	clone := *rel
	return &clone
}

func (s *releaseTestState) update(id int64, payload map[string]any) (*fakeRelease, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var current *fakeRelease
	var oldTag string
	for tag, rel := range s.releases {
		if rel.ID == id {
			current = rel
			oldTag = tag
			break
		}
	}
	if current == nil {
		return nil, false
	}
	if value, ok := payload["tag_name"].(string); ok && strings.TrimSpace(value) != "" {
		current.TagName = value
	}
	if value, ok := payload["name"].(string); ok {
		current.Name = value
	}
	if value, ok := payload["body"].(string); ok {
		current.Body = value
	}
	if value, ok := payload["draft"].(bool); ok {
		current.Draft = value
		if value {
			current.PublishedAt = nil
		} else if current.PublishedAt == nil {
			now := time.Now().UTC()
			current.PublishedAt = &now
		}
	}
	if value, ok := payload["prerelease"].(bool); ok {
		current.Prerelease = value
	}
	if current.TagName != oldTag {
		delete(s.releases, oldTag)
		s.releases[current.TagName] = current
	}
	clone := *current
	return &clone, true
}

func (s *releaseTestState) delete(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for tag, rel := range s.releases {
		if rel.ID == id {
			delete(s.releases, tag)
			return true
		}
	}
	return false
}

func releaseTestServer() *httptest.Server {
	state := newReleaseTestState()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v3/repos/owner/repo/releases" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(state.list())
		case r.URL.Path == "/api/v3/repos/owner/repo/releases" && r.Method == http.MethodPost:
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			release := state.create(
				asString(payload["tag_name"]),
				asString(payload["name"]),
				asString(payload["body"]),
				asBool(payload["draft"]),
				asBool(payload["prerelease"]),
			)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(release)
		case strings.HasPrefix(r.URL.Path, "/api/v3/repos/owner/repo/releases/tags/") && r.Method == http.MethodGet:
			tag := strings.TrimPrefix(r.URL.Path, "/api/v3/repos/owner/repo/releases/tags/")
			release, ok := state.getByTag(tag)
			if !ok {
				http.NotFound(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(release)
		case strings.HasPrefix(r.URL.Path, "/api/v3/repos/owner/repo/releases/"):
			idText := strings.TrimPrefix(r.URL.Path, "/api/v3/repos/owner/repo/releases/")
			id, err := strconv.ParseInt(idText, 10, 64)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			switch r.Method {
			case http.MethodPatch:
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				release, ok := state.update(id, payload)
				if !ok {
					http.NotFound(w, r)
					return
				}
				_ = json.NewEncoder(w).Encode(release)
			case http.MethodDelete:
				if !state.delete(id) {
					http.NotFound(w, r)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			default:
				http.NotFound(w, r)
			}
		case r.URL.Path == "/api/v3/repos/owner/repo/commits/refs/tags/v1.0.0/status":
			_, _ = w.Write([]byte(`{"state":"success","statuses":[{"context":"build","state":"success","description":"CI passed"}]}`))
		case r.URL.Path == "/api/v3/repos/owner/repo/commits/refs/tags/v1.0.0/check-runs":
			_, _ = w.Write([]byte(`{"total_count":1,"check_runs":[{"name":"tests","status":"completed","conclusion":"success"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func asString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func asBool(value any) bool {
	if b, ok := value.(bool); ok {
		return b
	}
	return false
}

func releaseRoot(t *testing.T, server *httptest.Server, args ...string) (*bytes.Buffer, error) {
	t.Helper()
	cfgDir := t.TempDir()
	t.Setenv("GITDEX_USER_CONFIG_DIR", cfgDir)
	restore := command.SetGitHubClientFactoryForTest(func(app bootstrap.App) (*ghclient.Client, error) {
		return ghclient.NewClientWithBaseURL(server.Client(), server.URL+"/api/v3")
	})
	defer restore()

	root := command.NewRootCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(args)
	return &stdout, root.Execute()
}

func TestReleaseAssessCommand_Registered(t *testing.T) {
	root := command.NewRootCommand()
	cmd, _, err := root.Find([]string{"release", "assess"})
	if err != nil {
		t.Fatalf("release assess not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("release assess command is nil")
	}
}

func TestReleaseAssessCommand_RequiresRepoAndTag(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("GITDEX_USER_CONFIG_DIR", cfgDir)

	root := command.NewRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"release", "assess"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when repo/tag missing")
	}
}

func TestReleaseAssessCommand_Runs(t *testing.T) {
	server := releaseTestServer()
	defer server.Close()
	stdout, err := releaseRoot(t, server, "release", "assess", "--repo", "owner/repo", "--tag", "v1.0.0")
	if err != nil {
		t.Fatalf("release assess failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "Release Readiness") && !strings.Contains(stdout.String(), "ready") {
		t.Errorf("unexpected output: %s", stdout.String())
	}
}

func TestReleaseAssessCommand_JSONOutput(t *testing.T) {
	server := releaseTestServer()
	defer server.Close()
	stdout, err := releaseRoot(t, server, "release", "assess", "--repo", "owner/repo", "--tag", "v1.0.0", "--output", "json")
	if err != nil {
		t.Fatalf("release assess failed: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, stdout.String())
	}
	for _, f := range []string{"repo_owner", "repo_name", "tag", "status", "assessed_at"} {
		if _, ok := result[f]; !ok {
			t.Errorf("JSON missing field %q: %v", f, result)
		}
	}
}

func TestReleaseListCommand_Registered(t *testing.T) {
	root := command.NewRootCommand()
	cmd, _, err := root.Find([]string{"release", "list"})
	if err != nil {
		t.Fatalf("release list not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("release list command is nil")
	}
}

func TestReleaseListCommand_Runs(t *testing.T) {
	server := releaseTestServer()
	defer server.Close()
	stdout, err := releaseRoot(t, server, "release", "list", "--repo", "owner/repo")
	if err != nil {
		t.Fatalf("release list failed: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "Recent Releases") && !strings.Contains(out, "v1.0.0") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestReleaseListCommand_JSONOutput(t *testing.T) {
	server := releaseTestServer()
	defer server.Close()
	stdout, err := releaseRoot(t, server, "release", "list", "--repo", "owner/repo", "--output", "json")
	if err != nil {
		t.Fatalf("release list failed: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, stdout.String())
	}
	if _, ok := result["releases"]; !ok {
		t.Errorf("JSON missing releases: %v", result)
	}
}

func TestReleaseLifecycleCommands(t *testing.T) {
	server := releaseTestServer()
	defer server.Close()
	if _, err := releaseRoot(t, server, "release", "create", "--repo", "owner/repo", "--tag", "v2.0.0", "--name", "Release 2", "--notes", "draft notes", "--draft"); err != nil {
		t.Fatalf("release create failed: %v", err)
	}

	stdout, err := releaseRoot(t, server, "release", "show", "--repo", "owner/repo", "--tag", "v2.0.0")
	if err != nil {
		t.Fatalf("release show failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "Release 2") {
		t.Fatalf("release show output missing updated name: %s", stdout.String())
	}

	if _, err := releaseRoot(t, server, "release", "edit", "--repo", "owner/repo", "--tag", "v2.0.0", "--name", "Release 2 GA", "--prerelease"); err != nil {
		t.Fatalf("release edit failed: %v", err)
	}

	stdout, err = releaseRoot(t, server, "release", "publish", "--repo", "owner/repo", "--tag", "v2.0.0")
	if err != nil {
		t.Fatalf("release publish failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "Draft: false") {
		t.Fatalf("release publish output missing draft=false: %s", stdout.String())
	}

	if _, err := releaseRoot(t, server, "release", "delete", "--repo", "owner/repo", "--tag", "v2.0.0"); err != nil {
		t.Fatalf("release delete failed: %v", err)
	}

	if _, err := releaseRoot(t, server, "release", "show", "--repo", "owner/repo", "--tag", "v2.0.0"); err == nil {
		t.Fatal("expected release show to fail after delete")
	}
}
