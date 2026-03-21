package presenter

import (
	"fmt"
	"io"
	"strings"

	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/tui/theme"
)

func RenderTextSummary(w io.Writer, s *repo.RepoSummary) error {
	if s == nil {
		_, err := fmt.Fprintln(w, "No repository data available")
		return err
	}

	token := theme.TokenForState(string(s.OverallLabel))

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Repository: %s/%s\n", s.Owner, s.Repo))
	b.WriteString(fmt.Sprintf("Overall:    %s %s\n", token.Icon, token.Label))
	b.WriteString(fmt.Sprintf("Timestamp:  %s\n", s.Timestamp.Format("2006-01-02 15:04:05 UTC")))
	b.WriteString("\n")

	b.WriteString("Dimensions:\n")
	b.WriteString(fmt.Sprintf("  %-15s %-12s %s\n", "Dimension", "Status", "Detail"))
	b.WriteString("  " + strings.Repeat("─", 55) + "\n")

	dims := []struct {
		name   string
		label  string
		detail string
	}{
		{"Local", string(s.Local.Label), s.Local.Detail},
		{"Remote", string(s.Remote.Label), s.Remote.Detail},
		{"Collaboration", string(s.Collaboration.Label), s.Collaboration.Detail},
		{"Workflows", string(s.Workflows.Label), s.Workflows.Detail},
		{"Deployments", string(s.Deployments.Label), s.Deployments.Detail},
	}

	for _, d := range dims {
		t := theme.TokenForState(d.label)
		detail := d.detail
		if detail == "" {
			detail = "-"
		}
		b.WriteString(fmt.Sprintf("  %-15s %s %-10s %s\n", d.name, t.Icon, t.Label, detail))
	}

	if len(s.Risks) > 0 {
		b.WriteString("\nRisks:\n")
		for _, r := range s.Risks {
			b.WriteString(fmt.Sprintf("  [%s] %s\n", r.Severity, r.Description))
			if r.Action != "" {
				b.WriteString(fmt.Sprintf("         → %s\n", r.Action))
			}
		}
	}

	if len(s.NextActions) > 0 {
		b.WriteString("\nNext Actions:\n")
		for _, a := range s.NextActions {
			b.WriteString(fmt.Sprintf("  → %s (%s)\n", a.Action, a.Reason))
		}
	}

	if len(s.Risks) == 0 && len(s.NextActions) == 0 {
		b.WriteString("\n✓ No material risks detected\n")
	}

	b.WriteString("\ngitdex> ")
	_, err := fmt.Fprint(w, b.String())
	return err
}
