package intent

type Source string

const (
	SourceCommand  Source = "command"
	SourceChat     Source = "chat"
	SourceTUI      Source = "tui"
	SourceAPI      Source = "api"
	SourceWebhook  Source = "webhook"
	SourceSchedule Source = "schedule"
)

type Intent struct {
	Source      Source            `json:"source" yaml:"source"`
	RawInput    string            `json:"raw_input" yaml:"raw_input"`
	ActionType  string            `json:"action_type" yaml:"action_type"`
	Parameters  map[string]string `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	ContextRefs []string          `json:"context_refs,omitempty" yaml:"context_refs,omitempty"`
}

func NewCommandIntent(rawInput, actionType string, params map[string]string) Intent {
	return Intent{
		Source:     SourceCommand,
		RawInput:   rawInput,
		ActionType: actionType,
		Parameters: params,
	}
}

func NewChatIntent(rawInput string) Intent {
	return Intent{
		Source:     SourceChat,
		RawInput:   rawInput,
		ActionType: "chat_derived",
	}
}
