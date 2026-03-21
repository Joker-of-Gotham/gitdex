package gitops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MirrorInfo holds metadata about a mirror repository.
type MirrorInfo struct {
	Owner     string    `json:"owner"`
	Repo      string    `json:"repo"`
	Path      string    `json:"path"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MirrorManager manages bare mirror repositories for fast clone/fetch.
type MirrorManager struct {
	executor  *GitExecutor
	mirrorDir string
}

// NewMirrorManager creates a new MirrorManager.
func NewMirrorManager(executor *GitExecutor, mirrorDir string) *MirrorManager {
	return &MirrorManager{executor: executor, mirrorDir: mirrorDir}
}

// MirrorPath returns the path for a mirror (owner/repo.git).
func (m *MirrorManager) MirrorPath(owner, repo string) string {
	return filepath.Join(m.mirrorDir, owner, repo+".git")
}

// EnsureMirror ensures a mirror exists and is up to date. If not exists, clones with --mirror. If exists, runs remote update.
func (m *MirrorManager) EnsureMirror(ctx context.Context, cloneURL string) (mirrorPath string, err error) {
	owner, repo, err := parseCloneURL(cloneURL)
	if err != nil {
		return "", err
	}
	mirrorPath = m.MirrorPath(owner, repo)

	if _, err := os.Stat(mirrorPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(mirrorPath), 0755); err != nil {
			return "", fmt.Errorf("create mirror parent dir: %w", err)
		}
		_, err = m.executor.Run(ctx, "", "clone", "--mirror", cloneURL, mirrorPath)
		if err != nil {
			return "", err
		}
		return mirrorPath, nil
	}

	_, err = m.executor.Run(ctx, mirrorPath, "remote", "update")
	if err != nil {
		return "", err
	}
	return mirrorPath, nil
}

// UpdateMirror runs git remote update --prune on the mirror.
func (m *MirrorManager) UpdateMirror(ctx context.Context, mirrorPath string) error {
	_, err := m.executor.Run(ctx, mirrorPath, "remote", "update", "--prune")
	return err
}

// ListMirrors walks mirrorDir to find all *.git directories and returns MirrorInfo.
func (m *MirrorManager) ListMirrors() ([]MirrorInfo, error) {
	var result []MirrorInfo
	err := filepath.Walk(m.mirrorDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".git") {
			rel, err := filepath.Rel(m.mirrorDir, path)
			if err != nil {
				return nil
			}
			parts := strings.Split(filepath.ToSlash(rel), "/")
			if len(parts) >= 2 {
				owner := parts[0]
				repo := strings.TrimSuffix(parts[len(parts)-1], ".git")
				var updatedAt time.Time
				if fi, err := os.Stat(filepath.Join(path, "HEAD")); err == nil {
					updatedAt = fi.ModTime()
				}
				result = append(result, MirrorInfo{
					Owner:     owner,
					Repo:      repo,
					Path:      path,
					UpdatedAt: updatedAt,
				})
			}
			return filepath.SkipDir
		}
		return nil
	})
	return result, err
}

// RemoveMirror deletes the mirror directory for owner/repo.
func (m *MirrorManager) RemoveMirror(ctx context.Context, owner, repo string) error {
	mirrorPath := m.MirrorPath(owner, repo)
	return os.RemoveAll(mirrorPath)
}

func parseCloneURL(url string) (owner, repo string, err error) {
	// Support https://github.com/owner/repo.git or git@github.com:owner/repo.git
	url = strings.TrimSuffix(url, ".git")
	var suffix string
	if idx := strings.LastIndex(url, "github.com/"); idx >= 0 {
		suffix = strings.TrimPrefix(url[idx:], "github.com/")
	} else if idx := strings.LastIndex(url, "github.com:"); idx >= 0 {
		suffix = strings.TrimPrefix(url[idx:], "github.com:")
	} else {
		return "", "", fmt.Errorf("unsupported clone URL format: %s", url)
	}
	parts := strings.Split(suffix, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("could not parse owner/repo from URL: %s", url)
	}
	return parts[0], parts[len(parts)-1], nil
}
