package dotgitdex

import (
	"fmt"
	"os"
	"strings"
)

// AppendCreativeProposal appends items to proposal/creative-proposal.md.
func (m *Manager) AppendCreativeProposal(items []string) error {
	return appendProposalFile(m.CreativeProposalPath(), "Creative-Proposal", items)
}

// AppendDiscardedProposal appends items to proposal/discarded-proposal.md.
func (m *Manager) AppendDiscardedProposal(items []string) error {
	return appendProposalFile(m.DiscardedProposalPath(), "Discarded-Proposal", items)
}

// ReadCreativeProposals reads all entries from creative-proposal.md.
func (m *Manager) ReadCreativeProposals() ([]string, error) {
	return readProposalFile(m.CreativeProposalPath())
}

// ReadDiscardedProposals reads all entries from discarded-proposal.md.
func (m *Manager) ReadDiscardedProposals() ([]string, error) {
	return readProposalFile(m.DiscardedProposalPath())
}

func appendProposalFile(path, heading string, items []string) error {
	if len(items) == 0 {
		return nil
	}
	existing, _ := readProposalFile(path)
	all := append(existing, items...)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s\n\n", heading))
	for _, item := range all {
		b.WriteString(fmt.Sprintf("- %s\n", item))
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func readProposalFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var items []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			items = append(items, strings.TrimPrefix(line, "- "))
		}
	}
	return items, nil
}
