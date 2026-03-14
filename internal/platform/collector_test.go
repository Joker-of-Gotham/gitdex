package platform

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseBitbucketWorkspaceRepo(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		workspace string
		repo      string
	}{
		{
			name:      "ssh",
			remoteURL: "git@bitbucket.org:team/sample-repo.git",
			workspace: "team",
			repo:      "sample-repo",
		},
		{
			name:      "https",
			remoteURL: "https://bitbucket.org/team/sample-repo.git",
			workspace: "team",
			repo:      "sample-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace, repo, err := parseBitbucketWorkspaceRepo(tt.remoteURL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if workspace != tt.workspace || repo != tt.repo {
				t.Fatalf("got %s/%s, want %s/%s", workspace, repo, tt.workspace, tt.repo)
			}
		})
	}
}

func TestNormalizeCIStatus(t *testing.T) {
	cases := map[string]string{
		"success": "passing",
		"passed":  "passing",
		"failed":  "failing",
		"running": "pending",
		"queued":  "unknown",
	}

	for in, want := range cases {
		if got := normalizeCIStatus(in); got != want {
			t.Fatalf("normalizeCIStatus(%q)=%q want %q", in, got, want)
		}
	}
}

func TestCollectorCollectSurfaceStates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/hooks":
			_, _ = w.Write([]byte(`[]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	collector := NewCollector("", "", "")
	collector.httpClient = server.Client()

	state := &PlatformState{}
	collector.collectSurfaceStates(t.Context(), PlatformBitbucket, server.URL, nil, state)
	if len(state.SurfaceStates) == 0 {
		t.Fatal("expected surface states")
	}
	found := false
	for _, item := range state.SurfaceStates {
		if item == "webhooks=available" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected webhooks probe to be available, got %v", state.SurfaceStates)
	}
}
