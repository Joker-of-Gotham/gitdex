package executor

import (
	"fmt"
	"runtime"
	"strings"
)

// Platform holds OS-specific shell and quoting conventions.
type Platform struct {
	OS       string
	Shell    string
	ShellArg string
	Quote    func(string) string
}

// DetectPlatform returns the platform configuration for the current OS.
func DetectPlatform() Platform {
	switch runtime.GOOS {
	case "windows":
		return Platform{
			OS:       "windows",
			Shell:    "cmd",
			ShellArg: "/C",
			Quote:    quoteWindows,
		}
	default:
		return Platform{
			OS:       runtime.GOOS,
			Shell:    "sh",
			ShellArg: "-c",
			Quote:    quoteUnix,
		}
	}
}

// quoteUnix wraps a string in single quotes, escaping internal single quotes.
func quoteUnix(s string) string {
	if !strings.ContainsAny(s, " \t\n'\"\\$!`(){}[]|&;<>?*~#") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// quoteWindows wraps a string in double quotes for cmd.exe.
func quoteWindows(s string) string {
	if !strings.ContainsAny(s, " \t\"&|<>^%") {
		return s
	}
	escaped := strings.ReplaceAll(s, `"`, `""`)
	return fmt.Sprintf(`"%s"`, escaped)
}

// IsWindows returns true if the current OS is Windows.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}
