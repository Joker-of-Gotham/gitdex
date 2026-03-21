package command

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	clioutput "github.com/your-org/gitdex/internal/cli/output"
)

type Capability struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Available   bool   `json:"available"`
}

type CapabilitiesList struct {
	Capabilities []Capability `json:"capabilities"`
}

func newCapabilitiesCommand(flags *runtimeOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "capabilities",
		Short: "List all available Gitdex capabilities",
		Long: `Display a structured list of all available Gitdex commands and capabilities
in the current environment. Use --output json for machine-readable output.`,
		Example: `  gitdex capabilities
  gitdex capabilities --output json
  gitdex capabilities --output yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			caps := discoverCapabilities(cmd.Root())

			format := effectiveOutputFormat(cmd, *flags, "")
			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, CapabilitiesList{Capabilities: caps})
			}

			return renderCapabilitiesText(cmd.OutOrStdout(), caps)
		},
	}
}

func discoverCapabilities(root *cobra.Command) []Capability {
	var caps []Capability
	walkCommands(root, &caps)
	return caps
}

func walkCommands(cmd *cobra.Command, caps *[]Capability) {
	for _, child := range cmd.Commands() {
		if child.Hidden {
			continue
		}
		if child.Name() == "help" || child.Name() == "completion" {
			continue
		}

		*caps = append(*caps, Capability{
			Command:     child.CommandPath(),
			Description: child.Short,
			Available:   child.Runnable(),
		})

		if child.HasSubCommands() {
			walkCommands(child, caps)
		}
	}
}

func renderCapabilitiesText(out io.Writer, caps []Capability) error {
	if _, err := fmt.Fprintln(out, "Available capabilities:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, ""); err != nil {
		return err
	}

	for _, cap := range caps {
		status := "✓"
		if !cap.Available {
			status = "○"
		}
		if _, err := fmt.Fprintf(out, "  %s %-30s %s\n", status, cap.Command, cap.Description); err != nil {
			return err
		}
	}

	return nil
}
