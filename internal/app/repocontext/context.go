package repocontext

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/gitops"
	ghclient "github.com/your-org/gitdex/internal/platform/github"
	"github.com/your-org/gitdex/internal/platform/identity"
)

type AccessMode string

const (
	AccessModeUnknown       AccessMode = "unknown"
	AccessModeRemoteOnly    AccessMode = "remote_only"
	AccessModeLocalWritable AccessMode = "local_writable"
)

type RemoteBinding struct {
	Name       string `json:"name" yaml:"name"`
	URL        string `json:"url" yaml:"url"`
	Normalized string `json:"normalized" yaml:"normalized"`
	Role       string `json:"role" yaml:"role"`
}

type RemoteTopology struct {
	Canonical RemoteBinding   `json:"canonical" yaml:"canonical"`
	Remotes   []RemoteBinding `json:"remotes" yaml:"remotes"`
}

type RepoContext struct {
	Host            string         `json:"host" yaml:"host"`
	Owner           string         `json:"owner" yaml:"owner"`
	Repo            string         `json:"repo" yaml:"repo"`
	CanonicalRemote string         `json:"canonical_remote,omitempty" yaml:"canonical_remote,omitempty"`
	LocalPaths      []string       `json:"local_paths,omitempty" yaml:"local_paths,omitempty"`
	ActiveLocalPath string         `json:"active_local_path,omitempty" yaml:"active_local_path,omitempty"`
	AccessMode      AccessMode     `json:"access_mode" yaml:"access_mode"`
	Topology        RemoteTopology `json:"topology" yaml:"topology"`
}

type ResolveOptions struct {
	RepoRoot   string
	Owner      string
	Repo       string
	RepoSpec   string
	RemoteURL  string
	LocalPath  string
	PreferHost string
}

func Resolve(ctx context.Context, app bootstrap.App, opts ResolveOptions) (*RepoContext, error) {
	repoRoot := cleanPath(firstNonEmpty(opts.LocalPath, opts.RepoRoot))
	host, owner, repoName := parseRepoInputs(opts)
	if repoRoot != "" {
		topology, topologyOwner, topologyRepo := inspectLocalTopology(ctx, repoRoot)
		if owner == "" {
			owner = topologyOwner
		}
		if repoName == "" {
			repoName = topologyRepo
		}
		if host == "" {
			host = hostFromTopology(topology)
		}
	}
	if host == "" {
		host = strings.TrimSpace(opts.PreferHost)
	}
	if host == "" {
		host = effectiveGitHubHost(app)
	}

	if owner == "" || repoName == "" {
		mode := AccessModeUnknown
		localPaths := []string{}
		if repoRoot != "" {
			mode = AccessModeLocalWritable
			localPaths = append(localPaths, filepath.ToSlash(repoRoot))
		}
		return &RepoContext{
			Host:            host,
			LocalPaths:      localPaths,
			ActiveLocalPath: repoRoot,
			AccessMode:      mode,
		}, nil
	}

	localPaths := discoverLocalPaths(ctx, app, host, owner, repoName)
	if repoRoot != "" {
		localPaths = appendUniquePaths(localPaths, repoRoot)
	}

	activeLocalPath := repoRoot
	if activeLocalPath == "" && len(localPaths) > 0 {
		activeLocalPath = localPaths[0]
	}

	topology, _, _ := inspectLocalTopology(ctx, activeLocalPath)
	canonicalRemote := topology.Canonical.Normalized
	if canonicalRemote == "" && opts.RemoteURL != "" {
		canonicalRemote = gitops.NormalizeRemoteURL(opts.RemoteURL)
	}
	if canonicalRemote == "" {
		canonicalRemote = strings.ToLower(host) + "/" + strings.ToLower(owner) + "/" + strings.ToLower(repoName)
	}

	mode := AccessModeRemoteOnly
	if activeLocalPath != "" {
		mode = AccessModeLocalWritable
	}

	return &RepoContext{
		Host:            host,
		Owner:           owner,
		Repo:            repoName,
		CanonicalRemote: canonicalRemote,
		LocalPaths:      localPaths,
		ActiveLocalPath: activeLocalPath,
		AccessMode:      mode,
		Topology:        topology,
	}, nil
}

func NewGitHubClient(app bootstrap.App) (*ghclient.Client, error) {
	tr, err := identity.ResolveTransport(app.Config.Identity, http.DefaultTransport)
	if err != nil {
		if errors.Is(err, identity.ErrNoIdentity) {
			return nil, nil
		}
		return nil, err
	}

	httpClient := &http.Client{Transport: tr.Transport}
	if tr.Host != "" && tr.Host != "github.com" {
		return ghclient.NewClientWithBaseURL(httpClient, fmt.Sprintf("https://%s/api/v3", tr.Host))
	}
	return ghclient.NewClient(httpClient), nil
}

func DefaultCloneDir(app bootstrap.App, repoName string) string {
	if len(app.Config.Git.WorkspaceRoots) > 0 && strings.TrimSpace(app.Config.Git.WorkspaceRoots[0]) != "" {
		return filepath.Join(app.Config.Git.WorkspaceRoots[0], repoName)
	}
	workingDir := strings.TrimSpace(app.Config.Paths.WorkingDir)
	if workingDir == "" {
		workingDir = "."
	}
	return filepath.Join(workingDir, repoName)
}

func ParseRepoFlag(repoFlag string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(repoFlag), "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func ResolveOwnerRepoFromLocalPath(ctx context.Context, repoRoot string) (string, string) {
	topology, owner, repoName := inspectLocalTopology(ctx, repoRoot)
	if owner != "" || repoName != "" {
		return owner, repoName
	}
	if topology.Canonical.URL != "" {
		_, owner, repoName = ParseRemoteURL(topology.Canonical.URL)
	}
	return owner, repoName
}

func ParseRemoteURL(rawURL string) (host, owner, repoName string) {
	normalized := gitops.NormalizeRemoteURL(rawURL)
	if normalized == "" {
		return "", "", ""
	}
	parts := strings.Split(normalized, "/")
	if len(parts) < 3 {
		return "", "", ""
	}
	return parts[0], parts[len(parts)-2], parts[len(parts)-1]
}

func discoverLocalPaths(ctx context.Context, app bootstrap.App, host, owner, repoName string) []string {
	idx := gitops.NewLocalIndex(gitops.NewGitExecutor())
	idx.BuildWithRoots(ctx, app.Config.Git.WorkspaceRoots)
	rawPaths := idx.LookupByRepo(host, owner, repoName)
	return normalizePaths(rawPaths)
}

func inspectLocalTopology(ctx context.Context, repoRoot string) (RemoteTopology, string, string) {
	repoRoot = cleanPath(repoRoot)
	if repoRoot == "" {
		return RemoteTopology{}, "", ""
	}
	exec := gitops.NewGitExecutor()
	remotesRes, err := exec.Run(ctx, repoRoot, "remote")
	if err != nil {
		return RemoteTopology{}, "", ""
	}

	topology := RemoteTopology{}
	for _, name := range strings.Split(remotesRes.Stdout, "\n") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		res, err := exec.Run(ctx, repoRoot, "remote", "get-url", name)
		if err != nil {
			continue
		}
		url := strings.TrimSpace(res.Stdout)
		if url == "" {
			continue
		}
		binding := RemoteBinding{
			Name:       name,
			URL:        url,
			Normalized: gitops.NormalizeRemoteURL(url),
			Role:       classifyRemoteRole(name),
		}
		topology.Remotes = append(topology.Remotes, binding)
	}
	for _, binding := range topology.Remotes {
		if binding.Name == "origin" {
			topology.Canonical = binding
			break
		}
	}
	if topology.Canonical.URL == "" && len(topology.Remotes) > 0 {
		topology.Canonical = topology.Remotes[0]
	}
	_, owner, repoName := ParseRemoteURL(topology.Canonical.URL)
	return topology, owner, repoName
}

func classifyRemoteRole(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "origin":
		return "origin"
	case "upstream":
		return "upstream"
	case "fork":
		return "fork"
	case "mirror":
		return "mirror"
	default:
		return "secondary"
	}
}

func hostFromTopology(topology RemoteTopology) string {
	if topology.Canonical.URL == "" {
		return ""
	}
	host, _, _ := ParseRemoteURL(topology.Canonical.URL)
	return host
}

func effectiveGitHubHost(app bootstrap.App) string {
	if tr, err := identity.ResolveTransport(app.Config.Identity, nil); err == nil && strings.TrimSpace(tr.Host) != "" {
		return strings.TrimSpace(tr.Host)
	}
	host := strings.TrimSpace(app.Config.Identity.GitHubApp.Host)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	}
	if host == "" {
		return "github.com"
	}
	return host
}

func normalizePaths(rawPaths []string) []string {
	paths := make([]string, 0, len(rawPaths))
	seen := make(map[string]bool, len(rawPaths))
	for _, p := range rawPaths {
		p = cleanPath(p)
		if p == "" {
			continue
		}
		key := strings.ToLower(filepath.ToSlash(p))
		if seen[key] {
			continue
		}
		seen[key] = true
		paths = append(paths, filepath.ToSlash(p))
	}
	return paths
}

func appendUniquePaths(paths []string, extras ...string) []string {
	return normalizePaths(append(append([]string{}, paths...), extras...))
}

func cleanPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	return filepath.Clean(p)
}

func parseRepoInputs(opts ResolveOptions) (host, owner, repoName string) {
	if opts.RepoSpec != "" {
		owner, repoName = ParseRepoFlag(opts.RepoSpec)
	}
	if owner == "" {
		owner = strings.TrimSpace(opts.Owner)
	}
	if repoName == "" {
		repoName = strings.TrimSpace(opts.Repo)
	}
	if opts.RemoteURL != "" {
		host, _, _ = ParseRemoteURL(opts.RemoteURL)
		if owner == "" || repoName == "" {
			_, remoteOwner, remoteRepo := ParseRemoteURL(opts.RemoteURL)
			if owner == "" {
				owner = remoteOwner
			}
			if repoName == "" {
				repoName = remoteRepo
			}
		}
	}
	return strings.TrimSpace(host), owner, repoName
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
