package views

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleSystem    MessageRole = "system"
	RoleAssistant MessageRole = "assistant"
	RoleInfo      MessageRole = "info"
	RoleError     MessageRole = "error"
)

type Message struct {
	Role      MessageRole
	Content   string
	Timestamp time.Time
}

type AppendMessageMsg struct {
	Message Message
}

type ChatView struct {
	t         *theme.Theme
	messages  []Message
	width     int
	height    int
	streaming bool
	vp        viewport.Model
}

func NewChatView(t *theme.Theme) *ChatView {
	return &ChatView{
		t: t,
		messages: []Message{
			{
				Role:      RoleSystem,
				Content:   "Gitdex chat is ready. Enter a command or ask for repository assistance. Use /help to inspect command coverage.",
				Timestamp: time.Now(),
			},
		},
	}
}

func (v *ChatView) ID() ID        { return ViewChat }
func (v *ChatView) Title() string { return "Chat" }
func (v *ChatView) Init() tea.Cmd { return nil }
func (v *ChatView) IsStreaming() bool {
	return v.streaming
}

func (v *ChatView) BeginStream() {
	v.streaming = true
	v.messages = append(v.messages, Message{
		Role:      RoleAssistant,
		Content:   "",
		Timestamp: time.Now(),
	})
}

func (v *ChatView) EndStream() {
	v.streaming = false
	if n := len(v.messages); n > 0 && v.messages[n-1].Role == RoleAssistant {
		content := strings.TrimSuffix(v.messages[n-1].Content, " ...")
		if strings.TrimSpace(content) == "" {
			v.messages = v.messages[:n-1]
		} else {
			v.messages[n-1].Content = content
		}
	}
}

func (v *ChatView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case AppendMessageMsg:
		v.messages = append(v.messages, msg.Message)
		v.scrollToBottom()
	case StreamChunkMsg:
		if len(v.messages) > 0 {
			last := &v.messages[len(v.messages)-1]
			if last.Role == RoleAssistant {
				last.Content = strings.TrimSuffix(last.Content, " ...")
				last.Content += msg.Content
				if !msg.Done {
					last.Content += " ..."
				}
			}
		}
		if msg.Done {
			v.streaming = false
		}
		v.scrollToBottom()
	case StreamErrorMsg:
		v.streaming = false
		if len(v.messages) > 0 {
			last := &v.messages[len(v.messages)-1]
			if last.Role == RoleAssistant {
				content := strings.TrimSuffix(last.Content, " ...")
				if strings.TrimSpace(content) == "" {
					v.messages = v.messages[:len(v.messages)-1]
				} else {
					last.Content = content
				}
			}
		}
		v.messages = append(v.messages, Message{
			Role:      RoleError,
			Content:   msg.Error.Error(),
			Timestamp: time.Now(),
		})
		v.scrollToBottom()
	case tea.KeyPressMsg:
		v.syncViewport()
		var cmd tea.Cmd
		v.vp, cmd = v.vp.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *ChatView) Render() string {
	if v.width < 10 || v.height < 3 {
		return ""
	}
	v.syncViewport()
	return v.vp.View()
}

func (v *ChatView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.vp = viewport.New(viewport.WithWidth(width), viewport.WithHeight(height))
	v.syncViewport()
}

func (v *ChatView) AppendMessage(m Message) {
	v.messages = append(v.messages, m)
	v.scrollToBottom()
}

func (v *ChatView) Messages() []Message { return v.messages }

func (v *ChatView) renderMessage(m Message, maxWidth int) string {
	panelWidth := maxWidth
	if panelWidth > 96 {
		panelWidth = 96
	}
	if panelWidth < 28 {
		panelWidth = 28
	}

	headerLeft, badgeBG := roleLabel(m.Role), v.t.Surface()
	switch m.Role {
	case RoleUser:
		badgeBG = v.t.Primary()
	case RoleAssistant:
		badgeBG = v.t.Secondary()
	case RoleSystem:
		badgeBG = v.t.Info()
	case RoleInfo:
		badgeBG = v.t.Highlight()
	case RoleError:
		badgeBG = v.t.Danger()
	}

	badge := lipgloss.NewStyle().
		Bold(true).
		Foreground(v.t.OnPrimary()).
		Background(badgeBG).
		Padding(0, 1).
		Render(headerLeft)
	ts := lipgloss.NewStyle().
		Foreground(v.t.Timestamp()).
		Render(m.Timestamp.Format("15:04:05"))

	contentWidth := panelWidth - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	body := strings.TrimSpace(m.Content)
	if body == "" && m.Role == RoleAssistant && v.streaming {
		body = "Awaiting stream output ..."
	}
	switch m.Role {
	case RoleAssistant:
		body = render.Markdown(body, contentWidth)
	default:
		body = render.FillBlock(body, contentWidth, lipgloss.NewStyle().Foreground(v.t.Fg()))
	}
	body = render.FillBlock(body, contentWidth, lipgloss.NewStyle())

	panelBG := v.t.Surface()
	border := v.t.BorderColor()
	if m.Role == RoleUser {
		panelBG = v.t.Selection()
		border = v.t.Primary()
	}
	if m.Role == RoleError {
		border = v.t.Danger()
	}
	if m.Role == RoleSystem {
		border = v.t.Info()
	}

	card := render.SurfacePanel(badge+"  "+ts+"\n\n"+body, panelWidth, panelBG, border)
	if m.Role == RoleUser && maxWidth > panelWidth {
		return lipgloss.PlaceHorizontal(maxWidth, lipgloss.Right, card)
	}
	return card
}

func roleLabel(r MessageRole) string {
	switch r {
	case RoleUser:
		return "USER"
	case RoleAssistant:
		return "GITDEX"
	case RoleSystem:
		return "SYSTEM"
	case RoleInfo:
		return "INFO"
	case RoleError:
		return "ERROR"
	default:
		return strings.ToUpper(string(r))
	}
}

func (v *ChatView) scrollToBottom() {
	v.syncViewport()
	v.vp.GotoBottom()
}

func (v *ChatView) syncViewport() {
	contentWidth := v.width - 2
	if contentWidth < 22 {
		contentWidth = 22
	}

	parts := make([]string, 0, len(v.messages)*2)
	for _, m := range v.messages {
		parts = append(parts, v.renderMessage(m, contentWidth))
		parts = append(parts, "")
	}
	v.vp.SetContent(strings.Join(parts, "\n"))
}
