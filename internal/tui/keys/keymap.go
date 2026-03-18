// Package keys provides centralized key binding management.
// Inspired by gh-dash's keys/keys.go.
package keys

// Binding describes a single key binding.
type Binding struct {
	Keys    []string
	Help    string
	Enabled func() bool
}

// NewBinding creates a new key binding.
func NewBinding(keys []string, help string) Binding {
	return Binding{Keys: keys, Help: help}
}

// MatchesKey checks if a key string matches this binding.
func (b Binding) MatchesKey(key string) bool {
	if b.Enabled != nil && !b.Enabled() {
		return false
	}
	for _, k := range b.Keys {
		if k == key {
			return true
		}
	}
	return false
}

// IsEnabled returns whether this binding is active.
func (b Binding) IsEnabled() bool {
	if b.Enabled == nil {
		return true
	}
	return b.Enabled()
}

// KeyMap defines all key bindings for the application.
type KeyMap struct {
	Up       Binding
	Down     Binding
	Left     Binding
	Right    Binding
	PageDown Binding
	PageUp   Binding
	Home     Binding
	End      Binding

	NextView Binding
	PrevView Binding

	TogglePreview Binding
	Refresh       Binding
	Help          Binding
	Quit          Binding
	Escape        Binding
	Enter         Binding
	Tab           Binding
	Search        Binding

	Accept     Binding
	RunAll     Binding
	Skip       Binding
	SwitchMode Binding

	Push     Binding
	Pull     Binding
	Fetch    Binding
	Merge    Binding
	Rebase   Binding
	Stash    Binding
	Checkout Binding
	Commit   Binding
	Amend    Binding
	Stage    Binding
	Unstage  Binding
	Discard  Binding

	CreateIssue Binding
	CreatePR    Binding
	Comment     Binding
	Approve     Binding
	Close       Binding
	MergePR     Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() *KeyMap {
	return &KeyMap{
		Up:       NewBinding([]string{"up", "k"}, "up"),
		Down:     NewBinding([]string{"down", "j"}, "down"),
		Left:     NewBinding([]string{"left", "h"}, "left"),
		Right:    NewBinding([]string{"right", "l"}, "right"),
		PageDown: NewBinding([]string{"pgdown", "ctrl+d"}, "page down"),
		PageUp:   NewBinding([]string{"pgup", "ctrl+u"}, "page up"),
		Home:     NewBinding([]string{"home", "g"}, "first"),
		End:      NewBinding([]string{"end", "G"}, "last"),

		NextView: NewBinding([]string{"tab"}, "next view"),
		PrevView: NewBinding([]string{"shift+tab"}, "prev view"),

		TogglePreview: NewBinding([]string{"p"}, "preview"),
		Refresh:       NewBinding([]string{"ctrl+r"}, "refresh"),
		Help:          NewBinding([]string{"?"}, "help"),
		Quit:          NewBinding([]string{"q", "ctrl+c"}, "quit"),
		Escape:        NewBinding([]string{"esc"}, "escape"),
		Enter:         NewBinding([]string{"enter"}, "select"),
		Tab:           NewBinding([]string{"tab"}, "tab"),
		Search:        NewBinding([]string{"/"}, "search"),

		Accept:     NewBinding([]string{"a"}, "accept"),
		RunAll:     NewBinding([]string{"A"}, "run all"),
		Skip:       NewBinding([]string{"s"}, "skip"),
		SwitchMode: NewBinding([]string{"m"}, "mode"),

		Push:     NewBinding([]string{"P"}, "push"),
		Pull:     NewBinding([]string{"p"}, "pull"),
		Fetch:    NewBinding([]string{"f"}, "fetch"),
		Merge:    NewBinding([]string{"M"}, "merge"),
		Rebase:   NewBinding([]string{"r"}, "rebase"),
		Stash:    NewBinding([]string{"S"}, "stash"),
		Checkout: NewBinding([]string{"c"}, "checkout"),
		Commit:   NewBinding([]string{"C"}, "commit"),
		Amend:    NewBinding([]string{"ctrl+a"}, "amend"),
		Stage:    NewBinding([]string{"space"}, "stage"),
		Unstage:  NewBinding([]string{"u"}, "unstage"),
		Discard:  NewBinding([]string{"d"}, "discard"),

		CreateIssue: NewBinding([]string{"c"}, "create issue"),
		CreatePR:    NewBinding([]string{"c"}, "create PR"),
		Comment:     NewBinding([]string{"C"}, "comment"),
		Approve:     NewBinding([]string{"a"}, "approve"),
		Close:       NewBinding([]string{"x"}, "close"),
		MergePR:     NewBinding([]string{"M"}, "merge PR"),
	}
}

// Keys is the global KeyMap singleton.
var Keys = DefaultKeyMap()
