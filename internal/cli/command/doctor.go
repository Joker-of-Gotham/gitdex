package command

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/doctor"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
	"github.com/your-org/gitdex/internal/platform/config"
)

func newDoctorCommand(flags *runtimeOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose configuration, repository context, connectivity, and required tooling",
		RunE: func(cmd *cobra.Command, args []string) error {
			configOptions := buildConfigOptions(*flags, cmd)
			report, err := doctor.Run(doctor.Options{
				ConfigOptions: configOptions,
			})
			if err != nil {
				return err
			}

			loadedOutput := configuredOutput(configOptions)
			format := effectiveOutputFormat(cmd, *flags, loadedOutput)
			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, report)
			}

			return renderDoctorText(cmd.OutOrStdout(), report)
		},
	}

	return markSkipBootstrap(cmd)
}

func configuredOutput(opts config.Options) string {
	cfg, err := config.Load(opts)
	if err != nil {
		return ""
	}
	return cfg.Output
}

func renderDoctorText(out io.Writer, report doctor.Report) error {
	if _, err := fmt.Fprintf(out, "Doctor status: %s\n", report.Status); err != nil {
		return err
	}
	for _, check := range report.Checks {
		if _, err := fmt.Fprintf(out, "\n[%s] %s\n", check.Status, check.ID); err != nil {
			return err
		}
		if err := renderKeyValueLine(out, "Summary", check.Summary); err != nil {
			return err
		}
		if err := renderKeyValueLine(out, "Detail", check.Detail); err != nil {
			return err
		}
		if err := renderKeyValueLine(out, "Fix", check.Fix); err != nil {
			return err
		}
		if err := renderKeyValueLine(out, "Source", check.Source); err != nil {
			return err
		}
	}
	return nil
}
