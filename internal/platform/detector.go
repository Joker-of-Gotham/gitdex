package platform

import (
	"os"
	"path/filepath"
	"strings"
)

// SSHKeyInfo holds info about a detected SSH key.
type SSHKeyInfo struct {
	Path    string
	Type    string // e.g. "rsa", "ed25519"
	HasPair bool   // true if .pub exists
}

// DetectSSHKeys checks ~/.ssh/ for id_rsa and id_ed25519, returns found keys.
func DetectSSHKeys() []SSHKeyInfo {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	sshDir := filepath.Join(home, ".ssh")
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil
	}

	var keys []SSHKeyInfo
	// Standard key names to look for
	want := map[string]string{
		"id_rsa":     "rsa",
		"id_ed25519": "ed25519",
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		for base, ktype := range want {
			if name == base {
				fullPath := filepath.Join(sshDir, base)
				_, err := os.Stat(fullPath)
				if err != nil {
					continue
				}
				pubPath := fullPath + ".pub"
				_, errPub := os.Stat(pubPath)
				keys = append(keys, SSHKeyInfo{
					Path:    fullPath,
					Type:    ktype,
					HasPair: errPub == nil,
				})
				break
			}
		}
	}
	return keys
}

// HasSSHKeys returns true if any SSH keys are detected.
func HasSSHKeys() bool {
	return len(DetectSSHKeys()) > 0
}

// DetectRemoteProtocol returns "ssh" or "https" based on URL format.
// SSH: git@host:user/repo or ssh://...
// HTTPS: https://... or http://...
func DetectRemoteProtocol(url string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return "https"
	}
	if strings.HasPrefix(url, "git@") || strings.HasPrefix(url, "ssh://") {
		return "ssh"
	}
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		return "https"
	}
	// Default to https for unknown format
	return "https"
}

// SuggestKeyGeneration returns a suggestion message if URL is SSH but no keys found.
func SuggestKeyGeneration(url string) string {
	if DetectRemoteProtocol(url) != "ssh" {
		return ""
	}
	if HasSSHKeys() {
		return ""
	}
	return "SSH URL detected but no SSH keys found. Run: ssh-keygen -t ed25519 -C \"your_email@example.com\""
}

// DetectPlatform identifies the hosting platform from a remote URL.
func DetectPlatform(remoteURL string) Platform {
	u := strings.ToLower(strings.TrimSpace(remoteURL))
	if strings.Contains(u, "github.com") {
		return PlatformGitHub
	}
	if strings.Contains(u, "gitlab") {
		return PlatformGitLab
	}
	if strings.Contains(u, "bitbucket") {
		return PlatformBitbucket
	}
	return PlatformUnknown
}
