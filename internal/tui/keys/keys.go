package keys

// Binding describes a single keybinding with display help.
type Binding struct {
	Keys []string
	Help string
	Desc string
}

// Enabled returns true if the binding has at least one key.
func (b Binding) Enabled() bool { return len(b.Keys) > 0 }

// MatchesKey returns true if the given key string matches any of the binding's keys.
func (b Binding) MatchesKey(k string) bool {
	for _, bk := range b.Keys {
		if bk == k {
			return true
		}
	}
	return false
}

// KeyMap defines all keyboard bindings for the GitDex TUI.
type KeyMap struct {
	Up         Binding
	Down       Binding
	Left       Binding
	Right      Binding
	FirstLine  Binding
	LastLine   Binding
	PageDown   Binding
	PageUp     Binding

	NextSection Binding
	PrevSection Binding

	TogglePreview Binding
	Refresh       Binding
	Help          Binding
	Quit          Binding
	Escape        Binding

	RunAll     Binding
	RunNext    Binding
	Accept     Binding
	Skip       Binding
	SwitchMode Binding
	SwitchFlow Binding

	Search Binding
	Enter  Binding
	Tab    Binding
}

// Keys is the global default keybinding set.
var Keys = &KeyMap{
	Up:    Binding{Keys: []string{"up", "k"}, Help: "k/↑", Desc: "up"},
	Down:  Binding{Keys: []string{"down", "j"}, Help: "j/↓", Desc: "down"},
	Left:  Binding{Keys: []string{"left", "h"}, Help: "h/←", Desc: "left"},
	Right: Binding{Keys: []string{"right", "l"}, Help: "l/→", Desc: "right"},

	FirstLine: Binding{Keys: []string{"g", "home"}, Help: "g/Home", Desc: "first"},
	LastLine:  Binding{Keys: []string{"G", "end"}, Help: "G/End", Desc: "last"},
	PageDown:  Binding{Keys: []string{"pgdown", "ctrl+d"}, Help: "PgDn", Desc: "page down"},
	PageUp:    Binding{Keys: []string{"pgup", "ctrl+u"}, Help: "PgUp", Desc: "page up"},

	NextSection: Binding{Keys: []string{"tab"}, Help: "Tab", Desc: "next section"},
	PrevSection: Binding{Keys: []string{"shift+tab"}, Help: "S-Tab", Desc: "prev section"},

	TogglePreview: Binding{Keys: []string{"p"}, Help: "p", Desc: "toggle preview"},
	Refresh:       Binding{Keys: []string{"r"}, Help: "r", Desc: "refresh"},
	Help:          Binding{Keys: []string{"?"}, Help: "?", Desc: "help"},
	Quit:          Binding{Keys: []string{"q", "ctrl+c"}, Help: "q", Desc: "quit"},
	Escape:        Binding{Keys: []string{"esc"}, Help: "Esc", Desc: "back"},

	RunAll:     Binding{Keys: []string{"A"}, Help: "A", Desc: "run all"},
	RunNext:    Binding{Keys: []string{"x"}, Help: "x", Desc: "run next"},
	Accept:     Binding{Keys: []string{"a"}, Help: "a", Desc: "accept"},
	Skip:       Binding{Keys: []string{"s"}, Help: "s", Desc: "skip"},
	SwitchMode: Binding{Keys: []string{"m"}, Help: "m", Desc: "switch mode"},
	SwitchFlow: Binding{Keys: []string{"f"}, Help: "f", Desc: "switch flow"},

	Search: Binding{Keys: []string{"/"}, Help: "/", Desc: "search"},
	Enter:  Binding{Keys: []string{"enter"}, Help: "Enter", Desc: "confirm"},
	Tab:    Binding{Keys: []string{"tab"}, Help: "Tab", Desc: "next"},
}

// NavigationHelp returns help bindings for navigation context.
func (k KeyMap) NavigationHelp() []Binding {
	return []Binding{
		k.Up, k.Down, k.PageUp, k.PageDown,
		k.NextSection, k.PrevSection,
		k.FirstLine, k.LastLine,
	}
}

// ActionHelp returns help bindings for action context.
func (k KeyMap) ActionHelp() []Binding {
	return []Binding{
		k.Accept, k.Skip, k.RunAll, k.RunNext,
		k.Refresh, k.TogglePreview,
	}
}

// GlobalHelp returns help bindings always visible.
func (k KeyMap) GlobalHelp() []Binding {
	return []Binding{
		k.Help, k.Quit, k.Escape,
		k.SwitchMode, k.SwitchFlow,
	}
}

// FullHelp returns all help bindings grouped.
func (k KeyMap) FullHelp() [][]Binding {
	return [][]Binding{
		k.NavigationHelp(),
		k.ActionHelp(),
		k.GlobalHelp(),
	}
}
