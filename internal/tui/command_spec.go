package tui

import (
	"fmt"
	"slices"
	"strings"
)

// ObjectAction is the normalized command semantic unit.
type ObjectAction struct {
	Object string
	Action string
	Arg    string
	Raw    string
}

func (oa ObjectAction) Key() string {
	return oa.Object + "." + oa.Action
}

var slashToObjectAction = map[string]ObjectAction{
	"/goal":     {Object: "goal", Action: "create"},
	"/run":      {Object: "suggestion", Action: "execute"},
	"/mode":     {Object: "mode", Action: "set"},
	"/config":   {Object: "config", Action: "set"},
	"/creative": {Object: "creative", Action: "run"},
	"/analyze":  {Object: "flow", Action: "analyze"},
	"/interval": {Object: "cruise", Action: "set_interval"},
	"/help":     {Object: "ui", Action: "help"},
	"/palette":  {Object: "ui", Action: "command_palette"},
	"/test":     {Object: "llm", Action: "probe"},
	"/clear":    {Object: "ui", Action: "clear_log"},
	"/failures": {Object: "metrics", Action: "failure_dashboard"},
	"/replay":   {Object: "execution", Action: "replay"},
}

// ParseObjectActionFromSlash maps slash commands into normalized object-action semantics.
func ParseObjectActionFromSlash(input string) (ObjectAction, error) {
	input = strings.TrimSpace(input)
	if input == "" || !strings.HasPrefix(input, "/") {
		return ObjectAction{}, fmt.Errorf("not a slash command")
	}
	parts := strings.SplitN(input, " ", 2)
	head := strings.ToLower(strings.TrimSpace(parts[0]))
	base, ok := slashToObjectAction[head]
	if !ok {
		return ObjectAction{}, fmt.Errorf("unknown slash command: %s", head)
	}
	arg := ""
	if len(parts) > 1 {
		arg = strings.TrimSpace(parts[1])
	}
	base.Arg = arg
	base.Raw = input
	return base, nil
}

// CommandPaletteItems exposes slash commands for the interactive command palette.
func CommandPaletteItems() []string {
	items := []string{
		"/goal <description>      create goal",
		"/run accept              execute next suggestion",
		"/run all                 execute all pending suggestions",
		"/mode manual|auto|cruise set runtime mode",
		"/config                  open config page",
		"/creative                run creative flow now",
		"/analyze                 trigger analysis round",
		"/interval <seconds>      set cruise interval",
		"/help                    open help overlay",
		"/palette                 open command palette",
		"/test                    run LLM connectivity probe",
		"/failures                show failure taxonomy dashboard",
		"/replay                  generate replay script from recent rounds",
		"/clear                   clear operation log",
	}
	slices.Sort(items)
	return items
}
