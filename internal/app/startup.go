package app

import (
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
)

// StartupInfo holds results from first-run and compatibility checks.
type StartupInfo struct {
	GitVersion   string // e.g. "2.43.0"
	GitAvailable bool
	AIStatus     string
	SystemLang   string // "zh", "en", "ja", etc.
	FirstRun     bool
}

// runStartupChecks runs Git, provider, and language detection.
func (a *Application) runStartupChecks(cfg *config.Config) StartupInfo {
	info := StartupInfo{FirstRun: isFirstRun()}
	info.GitVersion, info.GitAvailable = checkGit()
	info.AIStatus = checkLLMEnvironment(cfg)
	info.SystemLang = detectLanguage()
	return info
}

func isFirstRun() bool {
	dir, err := config.GlobalConfigDir()
	if err != nil {
		return true
	}
	if _, err := os.Stat(dir); err == nil {
		return false
	}
	legacyDir, legacyErr := config.LegacyGlobalConfigDir()
	if legacyErr == nil {
		if _, err := os.Stat(legacyDir); err == nil {
			return false
		}
	}
	return true
}

func checkGit() (version string, available bool) {
	cmd := exec.Command("git", "version")
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	// "git version 2.43.0.windows.1" or "git version 2.43.0"
	s := strings.TrimSpace(string(out))
	if idx := strings.Index(s, " "); idx >= 0 {
		version = strings.TrimSpace(s[idx+1:])
	} else {
		version = s
	}
	return version, true
}

func checkOllama() string {
	// 1. Try HTTP - if reachable, Ollama is running
	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{Timeout: 2 * time.Second}).DialContext,
		},
	}
	resp, err := client.Get("http://localhost:11434/api/version")
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			return "running"
		}
	}

	// 2. Check if ollama binary exists (installed but not running)
	if _, err := exec.LookPath("ollama"); err == nil {
		return "installed"
	}

	return "not_installed"
}

func checkLLMEnvironment(cfg *config.Config) string {
	if cfg == nil {
		return "disabled"
	}
	roleStatuses := make([]string, 0, 2)
	seen := map[string]bool{}

	roles := []config.ModelConfig{cfg.LLM.PrimaryRole()}
	if cfg.LLM.Secondary.Enabled {
		roles = append(roles, cfg.LLM.SecondaryRole())
	}
	for _, role := range roles {
		provider := config.RoleProvider(role)
		if seen[provider] {
			continue
		}
		seen[provider] = true
		switch provider {
		case "openai", "deepseek":
			if config.ResolveRoleAPIKey(role) == "" {
				roleStatuses = append(roleStatuses, provider+": missing API key")
			} else {
				roleStatuses = append(roleStatuses, provider+": configured")
			}
		default:
			roleStatuses = append(roleStatuses, "ollama:"+checkOllama())
		}
	}
	if len(roleStatuses) == 0 {
		return "disabled"
	}
	return strings.Join(roleStatuses, " | ")
}

func detectLanguage() string {
	for _, key := range []string{"LANG", "LC_ALL", "LANGUAGE"} {
		if v := os.Getenv(key); v != "" {
			lang := parseLang(v)
			if lang != "" {
				return lang
			}
		}
	}
	return "en"
}

func parseLang(s string) string {
	s = strings.TrimSpace(strings.Split(s, ":")[0])
	s = strings.TrimSpace(strings.Split(s, ".")[0])
	s = strings.ToLower(s)
	switch {
	case strings.HasPrefix(s, "zh"):
		return "zh"
	case strings.HasPrefix(s, "ja"):
		return "ja"
	case strings.HasPrefix(s, "ko"):
		return "ko"
	case strings.HasPrefix(s, "en"):
		return "en"
	case strings.HasPrefix(s, "fr"):
		return "fr"
	case strings.HasPrefix(s, "de"):
		return "de"
	case strings.HasPrefix(s, "es"):
		return "es"
	}
	return ""
}
