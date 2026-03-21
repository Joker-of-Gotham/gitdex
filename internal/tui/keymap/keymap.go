package keymap

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
)

type Binding struct {
	Keys []string
	Help string
}

func (b Binding) Matches(msg tea.KeyPressMsg) bool {
	k := msg.String()
	for _, key := range b.Keys {
		if key == k {
			return true
		}
	}
	return false
}

type GlobalKeys struct {
	Quit    Binding
	Help    Binding
	Back    Binding
	Refresh Binding

	CmdPalette      key.Binding // ctrl+p
	SwitchDashboard key.Binding // f1
	SwitchChat      key.Binding // f2
	SwitchExplorer  key.Binding // f3
	SwitchWorkspace key.Binding // f4
	SwitchSettings  key.Binding // f5
	SwitchReflog    key.Binding // f6
	FocusNav        key.Binding // ctrl+1
	FocusMain       key.Binding // ctrl+2
	FocusInspector  key.Binding // ctrl+3
	ToggleInspector key.Binding // ctrl+i
	CycleTheme      key.Binding // ctrl+t
}

type ListKeys struct {
	Up     Binding
	Down   Binding
	Select Binding
}

func DefaultGlobalKeys() GlobalKeys {
	return GlobalKeys{
		Quit:            Binding{Keys: []string{"q", "ctrl+c"}, Help: "quit"},
		Help:            Binding{Keys: []string{"?"}, Help: "help"},
		Back:            Binding{Keys: []string{"escape"}, Help: "back"},
		Refresh:         Binding{Keys: []string{"ctrl+r"}, Help: "refresh"},
		CmdPalette:      key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "command palette")),
		SwitchDashboard: key.NewBinding(key.WithKeys("f1"), key.WithHelp("f1", "dashboard")),
		SwitchChat:      key.NewBinding(key.WithKeys("f2"), key.WithHelp("f2", "chat")),
		SwitchExplorer:  key.NewBinding(key.WithKeys("f3"), key.WithHelp("f3", "explorer")),
		SwitchWorkspace: key.NewBinding(key.WithKeys("f4"), key.WithHelp("f4", "workspace")),
		SwitchSettings:  key.NewBinding(key.WithKeys("f5"), key.WithHelp("f5", "settings")),
		SwitchReflog:    key.NewBinding(key.WithKeys("f6"), key.WithHelp("f6", "reflog")),
		FocusNav:        key.NewBinding(key.WithKeys("ctrl+1"), key.WithHelp("ctrl+1", "focus nav")),
		FocusMain:       key.NewBinding(key.WithKeys("ctrl+2"), key.WithHelp("ctrl+2", "focus main")),
		FocusInspector:  key.NewBinding(key.WithKeys("ctrl+3"), key.WithHelp("ctrl+3", "focus inspector")),
		ToggleInspector: key.NewBinding(key.WithKeys("ctrl+i"), key.WithHelp("ctrl+i", "toggle inspector")),
		CycleTheme:      key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("ctrl+t", "cycle theme")),
	}
}

func DefaultListKeys() ListKeys {
	return ListKeys{
		Up:     Binding{Keys: []string{"up", "k"}, Help: "up"},
		Down:   Binding{Keys: []string{"down", "j"}, Help: "down"},
		Select: Binding{Keys: []string{"enter"}, Help: "select"},
	}
}

type HelpItem struct {
	Key  string
	Desc string
}

func GlobalHelpItems() []HelpItem {
	gk := DefaultGlobalKeys()
	return []HelpItem{
		{Key: gk.Quit.Keys[0], Desc: gk.Quit.Help},
		{Key: gk.Help.Keys[0], Desc: gk.Help.Help},
		{Key: gk.Back.Keys[0], Desc: gk.Back.Help},
	}
}
