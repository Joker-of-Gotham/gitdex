package theme

import (
	"os"
	"strings"
)

// IconSet holds icons for status, git, file, navigation, UI, panels, and decoration.
// The default set deliberately avoids wide emoji so layouts stay stable across terminals.
type IconSet struct {
	// Status
	Healthy, Drifting, Blocked, Degraded, Unknown string
	Running, Paused, Queued                       string
	// Git
	Branch, Commit, Merge, PullRequest, Issue, Tag, Diff, Stash string
	// File
	FileCode, FileConfig, FileDoc, FileTest string
	Folder, FolderOpen                      string
	// Navigation
	ChevronRight, ChevronDown                 string
	ArrowUp, ArrowDown, ArrowLeft, ArrowRight string
	Home, Back                                string
	// UI
	Spinner                               []string
	Check, Cross, Warning, Info, Question string
	Lock, Unlock, Eye, EyeOff             string
	// Panels
	Dashboard, Chat, Plan, Task, Evidence, Search, Settings, Help string
	// Decoration
	Separator, Dot, Diamond, Star, Fire, Rocket string
}

// NerdFontIcons uses Nerd Font codepoints (requires a patched font).
var NerdFontIcons = IconSet{
	Healthy:      "\uF058",
	Drifting:     "\uF071",
	Blocked:      "\uF06E",
	Degraded:     "\uF201",
	Unknown:      "\uF128",
	Running:      "\uF04B",
	Paused:       "\uF04C",
	Queued:       "\uF017",
	Branch:       "\uE0A0",
	Commit:       "\uE0A1",
	Merge:        "\uE0A2",
	PullRequest:  "\uE0A3",
	Issue:        "\uE0A4",
	Tag:          "\uE0A5",
	Diff:         "\uE0B0",
	Stash:        "\uE0C6",
	FileCode:     "\uE615",
	FileConfig:   "\uE615",
	FileDoc:      "\uE0A7",
	FileTest:     "\uE0A7",
	Folder:       "\uE5FF",
	FolderOpen:   "\uE5C8",
	ChevronRight: "\uE0B1",
	ChevronDown:  "\uE0B2",
	ArrowUp:      "\uF062",
	ArrowDown:    "\uF063",
	ArrowLeft:    "\uF060",
	ArrowRight:   "\uF061",
	Home:         "\uF015",
	Back:         "\uF104",
	Spinner: []string{
		"\u28F0", "\u28F1", "\u28F2", "\u28F3", "\u28F4",
		"\u28F5", "\u28F6", "\u28F7", "\u28F8", "\u28F9",
	},
	Check:     "\uF00C",
	Cross:     "\uF00D",
	Warning:   "\uF071",
	Info:      "\uF05A",
	Question:  "\uF128",
	Lock:      "\uF023",
	Unlock:    "\uF09C",
	Eye:       "\uF06E",
	EyeOff:    "\uF070",
	Dashboard: "\uF0E4",
	Chat:      "\uF0E6",
	Plan:      "\uF0F9",
	Task:      "\uF00C",
	Evidence:  "\uF02D",
	Search:    "\uF002",
	Settings:  "\uF013",
	Help:      "\uF059",
	Separator: "\u2502",
	Dot:       "\u2022",
	Diamond:   "\u25C6",
	Star:      "\u2605",
	Fire:      "\uF06D",
	Rocket:    "\uF135",
}

// UnicodeIcons uses plain ASCII-safe symbols so terminals without glyph support
// still render aligned layouts.
var UnicodeIcons = IconSet{
	Healthy:      "OK",
	Drifting:     "~",
	Blocked:      "!",
	Degraded:     "-",
	Unknown:      "?",
	Running:      ">",
	Paused:       "||",
	Queued:       "...",
	Branch:       "git",
	Commit:       "@",
	Merge:        "<>",
	PullRequest:  "PR",
	Issue:        "IS",
	Tag:          "tag",
	Diff:         "+/-",
	Stash:        "[]",
	FileCode:     "{}",
	FileConfig:   "cfg",
	FileDoc:      "doc",
	FileTest:     "tst",
	Folder:       "[+]",
	FolderOpen:   "[-]",
	ChevronRight: ">",
	ChevronDown:  "v",
	ArrowUp:      "^",
	ArrowDown:    "v",
	ArrowLeft:    "<",
	ArrowRight:   ">",
	Home:         "H",
	Back:         "<",
	Spinner:      []string{"-", "\\", "|", "/"},
	Check:        "ok",
	Cross:        "x",
	Warning:      "!",
	Info:         "i",
	Question:     "?",
	Lock:         "#",
	Unlock:       "~",
	Eye:          "o",
	EyeOff:       "x",
	Dashboard:    "db",
	Chat:         "ai",
	Plan:         "pl",
	Task:         "tk",
	Evidence:     "ev",
	Search:       "?",
	Settings:     "cfg",
	Help:         "?",
	Separator:    "|",
	Dot:          ".",
	Diamond:      "*",
	Star:         "*",
	Fire:         "hot",
	Rocket:       "go",
}

// Icons is the active icon set.
var Icons = UnicodeIcons

// SetNerdFont switches between Nerd Font and Unicode icon sets.
func SetNerdFont(enabled bool) {
	if enabled {
		Icons = NerdFontIcons
		return
	}
	Icons = UnicodeIcons
}

// DetectNerdFont checks GITDEX_NERD_FONT env var.
func DetectNerdFont() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("GITDEX_NERD_FONT")))
	switch v {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}
