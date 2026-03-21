package command

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/google/uuid"
	"github.com/your-org/gitdex/internal/api"
	"github.com/your-org/gitdex/internal/app/bootstrap"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
)

var defaultAPIRouter = api.NewMemoryAPIRouter()

func newAPIGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Submit and query structured intents, plans, and tasks via machine API",
	}
	cmd.AddCommand(newAPISubmitCommand(flags, appFn))
	cmd.AddCommand(newAPIEndpointsCommand(flags, appFn))
	cmd.AddCommand(newAPIQueryCommand(flags, appFn))
	cmd.AddCommand(newAPIGetCommand(flags, appFn))
	cmd.AddCommand(newAPIExchangeGroupCommand(flags, appFn))
	return cmd
}

func newAPISubmitCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var endpoint, payloadStr string
	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Submit structured intent, plan, or task via API",
		Long:  "Submit structured data via the machine API. Use --endpoint intents|plans|tasks and --payload JSON.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			endpoint = strings.TrimSpace(strings.ToLower(endpoint))
			if endpoint == "" {
				return fmt.Errorf("--endpoint is required (intents, plans, or tasks)")
			}

			var path string
			switch endpoint {
			case "intents":
				path = "/api/v1/intents"
			case "plans":
				path = "/api/v1/plans"
			case "tasks":
				path = "/api/v1/tasks"
			default:
				return fmt.Errorf("invalid endpoint %q; use intents, plans, or tasks", endpoint)
			}

			payload := json.RawMessage{}
			if payloadStr != "" {
				payload = json.RawMessage(payloadStr)
			}

			req := &api.APIRequest{
				RequestID:  uuid.New().String(),
				Endpoint:   path,
				Method:     "POST",
				Payload:    payload,
				APIVersion: api.APIVersion,
				Timestamp:  time.Now().UTC(),
			}

			resp, err := defaultAPIRouter.Handle(req)
			if err != nil {
				return err
			}

			if app.StorageProvider != nil && resp != nil && resp.StatusCode == 201 {
				if perr := api.PersistSubmitResponse(app.StorageProvider, path, resp); perr != nil {
					return perr
				}
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, resp)
			}
			return renderAPIResponseText(cmd.OutOrStdout(), resp)
		},
	}
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "Endpoint: intents, plans, or tasks")
	cmd.Flags().StringVar(&payloadStr, "payload", "{}", "JSON payload")
	_ = cmd.MarkFlagRequired("endpoint")
	return cmd
}

func newAPIEndpointsCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "endpoints",
		Short: "List available API endpoints",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			endpoints := defaultAPIRouter.ListEndpoints()

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{"endpoints": endpoints})
			}
			return renderEndpointsText(cmd.OutOrStdout(), endpoints)
		},
	}
}

func newAPIQueryCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var queryType, filterStr string
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query task, campaign, or audit state via API",
		Long:  "Query via API. Use --type tasks|campaigns|audit and optional --filter key=value.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			qr := api.BuildQueryRequest(queryType, filterStr)
			result, err := api.NewProviderQueryRouter(app.StorageProvider).Query(qr)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}
			return renderQueryResultText(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().StringVar(&queryType, "type", "tasks", "Query type: tasks, campaigns, audit")
	cmd.Flags().StringVar(&filterStr, "filter", "", "Filter as key=value (e.g. status=running)")
	return cmd
}

func newAPIGetCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var endpoint, id string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get specific resource by ID via API",
		Long:  "Get a task or campaign by ID. Use --endpoint tasks|campaigns and --id <id>.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			if endpoint == "" || id == "" {
				return fmt.Errorf("--endpoint and --id are required")
			}

			result, err := api.NewProviderQueryRouter(app.StorageProvider).GetResource(endpoint, id)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}
			if len(result.Errors) > 0 {
				for _, e := range result.Errors {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Error: %s - %s\n", e.Code, e.Message)
				}
				return nil
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", string(result.Payload))
			return err
		},
	}
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "Resource: tasks or campaigns")
	cmd.Flags().StringVar(&id, "id", "", "Resource ID")
	_ = cmd.MarkFlagRequired("endpoint")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newAPIExchangeGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exchange",
		Short: "Import, export, and validate versioned exchange data",
	}
	cmd.AddCommand(newAPIExchangeImportCommand(flags, appFn))
	cmd.AddCommand(newAPIExchangeExportCommand(flags, appFn))
	cmd.AddCommand(newAPIExchangeValidateCommand(flags, appFn))
	return cmd
}

func newAPIExchangeImportCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var filePath, formatStr string
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import external data from file",
		Long:  "Import exchange payload from file. Use --file <path> and --format json|yaml.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			fmtOut := effectiveOutputFormat(cmd, *flags, app.Config.Output)
			if filePath == "" {
				return fmt.Errorf("--file is required")
			}
			payload, err := api.ReadExchangeFile(filePath)
			if err != nil {
				return err
			}
			if clioutput.IsStructured(fmtOut) {
				return clioutput.WriteValue(cmd.OutOrStdout(), fmtOut, payload)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Imported: %s (format=%s)\n", filePath, payload.Format)
			return nil
		},
	}
	cmd.Flags().StringVar(&filePath, "file", "", "Path to exchange file")
	cmd.Flags().StringVar(&formatStr, "format", "json", "Format: json, yaml")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func newAPIExchangeExportCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var exportType, formatStr string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export data for external tooling",
		Long:  "Export plans, results, or status. Use --type plans and --format json|yaml.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			fmtOut := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			var data json.RawMessage
			if app.StorageProvider != nil {
				engine := api.NewProviderExportEngine(app.StorageProvider)
				result, err := engine.Export(cmd.Context(), &api.ExportRequest{
					ExportType: api.ExportType(strings.TrimSpace(strings.ToLower(exportType))),
					Format:     strings.TrimSpace(strings.ToLower(formatStr)),
				})
				if err == nil {
					data = json.RawMessage(result.Data)
				} else {
					data = json.RawMessage(fmt.Sprintf(`{"error":%q}`, err.Error()))
				}
			} else {
				data = json.RawMessage(`{"error":"no storage provider configured"}`)
			}

			payload := &api.ExchangePayload{
				Format:        api.ExchangeFormat(strings.TrimSpace(strings.ToLower(formatStr))),
				APIVersion:    api.APIVersion,
				SchemaVersion: "1",
				PayloadType:   exportType,
				Data:          data,
				CreatedAt:     time.Now().UTC(),
			}
			if payload.Format == "" {
				payload.Format = api.ExchangeFormatJSON
			}
			if clioutput.IsStructured(fmtOut) {
				return clioutput.WriteValue(cmd.OutOrStdout(), fmtOut, payload)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Exported type=%s format=%s\n", exportType, payload.Format)
			return nil
		},
	}
	cmd.Flags().StringVar(&exportType, "type", "plans", "Export type: plans, results, status")
	cmd.Flags().StringVar(&formatStr, "format", "json", "Format: json, yaml")
	return cmd
}

func newAPIExchangeValidateCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	var filePath string
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate exchange payload file",
		Long:  "Validate format, version, and required fields of an exchange payload.",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = appFn()
			if filePath == "" {
				return fmt.Errorf("--file is required")
			}
			payload, err := api.ReadExchangeFile(filePath)
			if err != nil {
				return err
			}
			validator := api.NewDefaultExchangeValidator()
			if err := validator.Validate(payload); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Valid: %s\n", filePath)
			return nil
		},
	}
	cmd.Flags().StringVar(&filePath, "file", "", "Path to exchange file")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func renderAPIResponseText(out io.Writer, resp *api.APIResponse) error {
	_, _ = fmt.Fprintf(out, "Status: %d\n", resp.StatusCode)
	if len(resp.Errors) > 0 {
		for _, e := range resp.Errors {
			_, _ = fmt.Fprintf(out, "Error: %s - %s\n", e.Code, e.Message)
		}
	}
	if len(resp.Payload) > 0 {
		_, _ = fmt.Fprintf(out, "Payload: %s\n", string(resp.Payload))
	}
	return nil
}

func renderEndpointsText(out io.Writer, endpoints []api.Endpoint) error {
	for _, e := range endpoints {
		_, _ = fmt.Fprintf(out, "%s %s - %s\n", e.Method, e.Path, e.Description)
	}
	return nil
}

func renderQueryResultText(out io.Writer, result *api.QueryResult) error {
	_, _ = fmt.Fprintf(out, "Query type: %s\n", result.QueryType)
	_, _ = fmt.Fprintf(out, "Total: %d, Page: %d, Per page: %d\n", result.TotalCount, result.Page, result.PerPage)
	for i, item := range result.Items {
		_, _ = fmt.Fprintf(out, "[%d] %s\n", i+1, string(item))
	}
	return nil
}
