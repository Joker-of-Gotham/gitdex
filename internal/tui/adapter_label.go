package tui

import (
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

func adapterDisplayLabel(kind platform.AdapterKind) string {
	switch kind {
	case platform.AdapterAPI:
		return "api-backed"
	case platform.AdapterCLI:
		return "gh-backed"
	case platform.AdapterBrowser:
		return "browser-backed"
	default:
		return strings.TrimSpace(string(kind))
	}
}
