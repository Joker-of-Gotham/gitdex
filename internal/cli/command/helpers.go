package command

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	clioutput "github.com/your-org/gitdex/internal/cli/output"
	"github.com/your-org/gitdex/internal/platform/config"
)

const skipBootstrapAnnotation = "gitdex/skip-bootstrap"

func markSkipBootstrap(cmd *cobra.Command) *cobra.Command {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[skipBootstrapAnnotation] = "true"
	return cmd
}

func shouldSkipBootstrap(cmd *cobra.Command) bool {
	for current := cmd; current != nil; current = current.Parent() {
		if current.Annotations != nil && current.Annotations[skipBootstrapAnnotation] == "true" {
			return true
		}
	}

	return cmd != nil && cmd.Name() == "help"
}

func buildConfigOptions(flags runtimeOptions, cmd *cobra.Command) config.Options {
	return config.Options{
		ConfigFile:  flags.configFile,
		Output:      flags.output,
		OutputSet:   commandFlagChanged(cmd, "output"),
		LogLevel:    flags.logLevel,
		LogLevelSet: commandFlagChanged(cmd, "log-level"),
		Profile:     flags.profile,
		ProfileSet:  commandFlagChanged(cmd, "profile"),
	}
}

func effectiveOutputFormat(cmd *cobra.Command, flags runtimeOptions, loadedOutput string) string {
	if commandFlagChanged(cmd, "output") {
		return clioutput.Normalize(flags.output)
	}
	if value, ok := os.LookupEnv("GITDEX_OUTPUT"); ok && strings.TrimSpace(value) != "" {
		return clioutput.Normalize(value)
	}
	if strings.TrimSpace(loadedOutput) != "" {
		return clioutput.Normalize(loadedOutput)
	}
	return clioutput.FormatText
}

func renderKeyValueLine(out io.Writer, label, value string) error {
	_, err := fmt.Fprintf(out, "%s: %s\n", label, value)
	return err
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
