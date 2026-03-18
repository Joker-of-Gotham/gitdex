package executor

import (
	"os/exec"
	"runtime"
	"strings"
)

// Platform provides OS-specific command execution context.
// Inspired by lazygit's oscommands.Platform.
type Platform struct {
	OS       string
	Shell    string
	ShellArg string
	GitBin   string
	GhBin    string
}

// DetectPlatform detects the current OS and resolves tool binaries.
// Zero hardcoded paths — everything is resolved at runtime.
func DetectPlatform() *Platform {
	p := &Platform{
		OS: runtime.GOOS,
	}

	switch p.OS {
	case "windows":
		p.Shell = detectWindowsShell()
		if strings.Contains(strings.ToLower(p.Shell), "powershell") ||
			strings.Contains(strings.ToLower(p.Shell), "pwsh") {
			p.ShellArg = "-Command"
		} else {
			p.ShellArg = "/C"
		}
	default:
		p.Shell = detectUnixShell()
		p.ShellArg = "-c"
	}

	p.GitBin = resolveBinary("git")
	p.GhBin = resolveBinary("gh")

	return p
}

func detectWindowsShell() string {
	if path, err := exec.LookPath("pwsh"); err == nil {
		return path
	}
	if path, err := exec.LookPath("powershell"); err == nil {
		return path
	}
	return "cmd"
}

func detectUnixShell() string {
	for _, sh := range []string{"zsh", "bash", "sh"} {
		if path, err := exec.LookPath(sh); err == nil {
			return path
		}
	}
	return "/bin/sh"
}

func resolveBinary(name string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	return name
}

// IsWindows returns true if the current OS is Windows.
func (p *Platform) IsWindows() bool {
	return p.OS == "windows"
}

// Quote wraps a string in platform-appropriate quotes for shell use.
func (p *Platform) Quote(s string) string {
	if p.IsWindows() {
		if strings.Contains(strings.ToLower(p.Shell), "powershell") ||
			strings.Contains(strings.ToLower(p.Shell), "pwsh") {
			return "'" + strings.ReplaceAll(s, "'", "''") + "'"
		}
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// CrossPlatformCommands lists commands available on all platforms.
var CrossPlatformCommands = map[string]bool{
	"go": true, "make": true, "cmake": true, "gradle": true, "mvn": true,
	"npm": true, "npx": true, "yarn": true, "pnpm": true,
	"node": true, "deno": true, "bun": true, "tsc": true,
	"python": true, "python3": true, "pip": true, "pip3": true, "uv": true,
	"cargo": true, "rustc": true,
	"dotnet": true, "java": true, "javac": true,
	"docker": true, "docker-compose": true, "podman": true,
	"echo": true, "cp": true, "mv": true, "mkdir": true,
	"curl": true, "zip": true, "unzip": true,
	"tree": true, "rg": true,
}

// UnixOnlyCommands are only available on Unix/Linux/macOS.
var UnixOnlyCommands = map[string]bool{
	"cat": true, "ls": true, "head": true, "tail": true,
	"grep": true, "find": true, "wc": true, "sort": true, "uniq": true,
	"touch": true, "chmod": true, "chown": true, "ln": true,
	"wget": true, "tar": true, "gzip": true, "gunzip": true, "xz": true,
	"sed": true, "awk": true, "perl": true, "tr": true, "cut": true,
	"diff": true, "patch": true, "tee": true, "xargs": true,
	"which": true, "test": true, "true": true, "false": true,
	"env": true, "printenv": true,
	"rm": true, "rmdir": true, "realpath": true, "basename": true, "dirname": true,
}

// WindowsOnlyCommands are only available on Windows.
var WindowsOnlyCommands = map[string]bool{
	"dir": true, "where": true, "type": true, "set": true,
	"copy": true, "xcopy": true, "robocopy": true, "del": true,
	"rename": true, "ren": true, "move": true,
	"icacls": true, "attrib": true, "mklink": true,
	"findstr": true, "more": true,
	"certutil": true, "clip": true,
}

// ToolAlternatives suggests cross-platform alternatives for Unix-only tools.
var ToolAlternatives = map[string]string{
	"sed":      "Use file_read + file_write to modify file content",
	"awk":      "Use file_read + file_write to modify file content",
	"perl":     "Use file_read + file_write to modify file content",
	"tr":       "Use file_read + file_write to modify file content",
	"cut":      "Use file_read + file_write to modify file content",
	"head":     "Use file_read to read the file",
	"tail":     "Use file_read to read the file",
	"cat":      "Use file_read to read file contents",
	"ls":       "Use 'dir' on Windows",
	"find":     "Use 'dir /s /b' or 'where' on Windows",
	"grep":     "Use 'rg' (ripgrep) which is cross-platform",
	"touch":    "Use file_write with 'create' operation",
	"chmod":    "Use 'icacls' on Windows or skip this step",
	"chown":    "Not applicable on Windows; skip this step",
	"ln":       "Use 'mklink' on Windows",
	"wc":       "Use file_read + count, or a cross-platform tool",
	"sort":     "Use file_read + file_write to sort",
	"uniq":     "Use file_read + file_write to deduplicate",
	"wget":     "Use 'curl' which is available on Windows",
	"tar":      "Use 'zip'/'unzip' on Windows",
	"gzip":     "Use 'zip'/'unzip' on Windows",
	"gunzip":   "Use 'zip'/'unzip' on Windows",
	"xz":       "Use 'zip'/'unzip' on Windows",
	"diff":     "Use 'git diff' instead",
	"patch":    "Use 'git apply' instead",
	"tee":      "Use file_write with 'append' operation",
	"xargs":    "Use a loop or file_write",
	"which":    "Use 'where' on Windows",
	"env":      "Use 'set' on Windows",
	"printenv": "Use 'set' on Windows",
	"rm":       "Use 'del' on Windows or file_write with 'delete' operation",
	"rmdir":    "Use 'rmdir' (also works on Windows)",
	"realpath": "Not needed; use relative paths or file_read",
	"basename": "Not needed on Windows",
	"dirname":  "Not needed on Windows",
}

// IsCommandAllowed checks if a command binary is allowed on the current platform.
func (p *Platform) IsCommandAllowed(base string) bool {
	if CrossPlatformCommands[base] {
		return true
	}
	if p.IsWindows() {
		return WindowsOnlyCommands[base]
	}
	return UnixOnlyCommands[base]
}

// AlternativeHint returns a helpful suggestion for unavailable commands.
func AlternativeHint(base string) string {
	if hint, ok := ToolAlternatives[base]; ok {
		return hint
	}
	return ""
}
