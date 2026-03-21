package render

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/your-org/gitdex/internal/tui/theme"
)

func Code(source, filename string, width int, t *theme.Theme) string {
	lang := DetectLanguage(filename)
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	styleName := "monokai"
	if t != nil && !t.IsDark {
		styleName = "github"
	}
	style := styles.Get(styleName)
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		return plainCodeFallback(source, width)
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return plainCodeFallback(source, width)
	}

	lines := strings.Split(buf.String(), "\n")
	numWidth := len(fmt.Sprintf("%d", len(lines)))
	if numWidth < 3 {
		numWidth = 3
	}

	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#334155"))
	if t != nil {
		numStyle = lipgloss.NewStyle().Foreground(t.Timestamp())
		sepStyle = lipgloss.NewStyle().Foreground(t.Divider())
	}

	var out strings.Builder
	for i, line := range lines {
		num := numStyle.Render(fmt.Sprintf("%*d", numWidth, i+1))
		sep := sepStyle.Render(" | ")
		out.WriteString(num + sep + line + "\n")
	}
	return out.String()
}

const chunkedHighlightThreshold = 5000

func CodeChunked(source, filename string, width int, t *theme.Theme, scrollPct float64, vpHeight int) string {
	lines := strings.Split(source, "\n")
	totalLines := len(lines)

	if totalLines <= chunkedHighlightThreshold {
		return Code(source, filename, width, t)
	}

	vpOffset := int(scrollPct * float64(totalLines))
	bufStart := vpOffset - 100
	if bufStart < 0 {
		bufStart = 0
	}
	bufEnd := vpOffset + vpHeight + 100
	if bufEnd > totalLines {
		bufEnd = totalLines
	}

	chunk := strings.Join(lines[bufStart:bufEnd], "\n")
	highlighted := highlightChunk(chunk, filename, t)

	numWidth := len(fmt.Sprintf("%d", totalLines))
	if numWidth < 3 {
		numWidth = 3
	}
	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#334155"))
	if t != nil {
		numStyle = lipgloss.NewStyle().Foreground(t.Timestamp())
		sepStyle = lipgloss.NewStyle().Foreground(t.Divider())
	}

	var out strings.Builder
	for i := 0; i < totalLines; i++ {
		num := numStyle.Render(fmt.Sprintf("%*d", numWidth, i+1))
		sep := sepStyle.Render(" | ")
		if i >= bufStart && i < bufEnd {
			hlIdx := i - bufStart
			hlLines := strings.Split(highlighted, "\n")
			if hlIdx < len(hlLines) {
				out.WriteString(num + sep + hlLines[hlIdx] + "\n")
			} else {
				out.WriteString(num + sep + lines[i] + "\n")
			}
		} else {
			out.WriteString(num + sep + lines[i] + "\n")
		}
	}
	return out.String()
}

func highlightChunk(source, filename string, t *theme.Theme) string {
	lang := DetectLanguage(filename)
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	styleName := "monokai"
	if t != nil && !t.IsDark {
		styleName = "github"
	}
	style := styles.Get(styleName)
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}
	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		return source
	}
	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return source
	}
	return buf.String()
}

func Markdown(md string, width int) string {
	if width < 20 {
		width = 80
	}
	r, err := glamour.NewTermRenderer(glamour.WithWordWrap(width))
	if err != nil {
		return md
	}
	out, err := r.Render(md)
	if err != nil {
		return md
	}
	return strings.TrimRight(out, "\n")
}

func Diff(diff string, t *theme.Theme) string {
	if diff == "" {
		return ""
	}

	addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#16A34A"))
	delStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#DC2626"))
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#2563EB")).Bold(true)
	hunkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#0891B2"))
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))
	if t != nil {
		addStyle = lipgloss.NewStyle().Foreground(t.Success())
		delStyle = lipgloss.NewStyle().Foreground(t.Danger())
		headerStyle = lipgloss.NewStyle().Foreground(t.Primary()).Bold(true)
		hunkStyle = lipgloss.NewStyle().Foreground(t.Info())
		normalStyle = lipgloss.NewStyle().Foreground(t.Fg())
	}

	var out strings.Builder
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"), strings.HasPrefix(line, "diff "):
			out.WriteString(headerStyle.Render(line))
		case strings.HasPrefix(line, "@@"):
			out.WriteString(hunkStyle.Render(line))
		case strings.HasPrefix(line, "+"):
			out.WriteString(addStyle.Render(line))
		case strings.HasPrefix(line, "-"):
			out.WriteString(delStyle.Render(line))
		default:
			out.WriteString(normalStyle.Render(line))
		}
		out.WriteString("\n")
	}
	return out.String()
}

func DetectLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	base := strings.ToLower(filepath.Base(filename))

	switch base {
	case "dockerfile":
		return "docker"
	case "makefile", "gnumakefile":
		return "make"
	case ".gitignore", ".dockerignore":
		return "bash"
	case "go.mod", "go.sum":
		return "go"
	}

	switch ext {
	case ".go":
		return "go"
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".rb":
		return "ruby"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".swift":
		return "swift"
	case ".kt", ".kts":
		return "kotlin"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".xml":
		return "xml"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	case ".scss":
		return "scss"
	case ".less":
		return "less"
	case ".sql":
		return "sql"
	case ".sh", ".bash", ".zsh":
		return "bash"
	case ".ps1":
		return "powershell"
	case ".md", ".markdown":
		return "markdown"
	case ".lua":
		return "lua"
	case ".php":
		return "php"
	case ".r":
		return "r"
	case ".scala":
		return "scala"
	case ".ex", ".exs":
		return "elixir"
	case ".hs":
		return "haskell"
	case ".vim":
		return "vim"
	case ".tf":
		return "hcl"
	case ".proto":
		return "protobuf"
	case ".graphql", ".gql":
		return "graphql"
	default:
		return ""
	}
}

func plainCodeFallback(source string, width int) string {
	lines := strings.Split(source, "\n")
	numWidth := len(fmt.Sprintf("%d", len(lines)))
	if numWidth < 3 {
		numWidth = 3
	}

	var out strings.Builder
	for i, line := range lines {
		out.WriteString(fmt.Sprintf("%*d | %s\n", numWidth, i+1, line))
	}
	return out.String()
}
