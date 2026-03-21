package command

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/autonomy"
)

type daemonAutomation struct {
	scheduler *autonomy.Scheduler
	cron      *cron.Cron
}

func (a *daemonAutomation) Stop() {
	if a == nil {
		return
	}
	if a.scheduler != nil {
		a.scheduler.Stop()
	}
	if a.cron != nil {
		ctx := a.cron.Stop()
		<-ctx.Done()
	}
}

func startDaemonAutomation(ctx context.Context, cmd *cobra.Command, app bootstrap.App) (*daemonAutomation, error) {
	automation := &daemonAutomation{
		scheduler: autonomy.NewScheduler(),
		cron: cron.New(cron.WithParser(cron.NewParser(
			cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
		))),
	}

	monitorConfigs, err := app.StorageProvider.MonitorStore().ListMonitorConfigs()
	if err != nil {
		return nil, fmt.Errorf("list monitor configs: %w", err)
	}
	for _, cfg := range monitorConfigs {
		if cfg == nil || !cfg.Enabled {
			continue
		}
		monitorCfg := *cfg
		if interval, ok := parseDaemonIntervalPattern(monitorCfg.Interval); ok {
			automation.scheduler.Register(autonomy.SchedulerTask{
				Name:     "monitor:" + monitorCfg.MonitorID,
				Interval: interval,
				Action: func(runCtx context.Context) error {
					_, err := runMonitorCheck(runCtx, cmd, app, app.StorageProvider.MonitorStore(), &monitorCfg, true, "")
					return err
				},
			})
			continue
		}
		if schedule, ok := parseDaemonCronPattern(monitorCfg.Interval); ok {
			if _, err := automation.cron.AddFunc(schedule, func() {
				_, _ = runMonitorCheck(ctx, cmd, app, app.StorageProvider.MonitorStore(), &monitorCfg, true, "")
			}); err != nil {
				return nil, fmt.Errorf("register monitor cron %s: %w", monitorCfg.MonitorID, err)
			}
		}
	}

	triggerConfigs, err := app.StorageProvider.TriggerStore().ListTriggers()
	if err != nil {
		return nil, fmt.Errorf("list trigger configs: %w", err)
	}
	for _, cfg := range triggerConfigs {
		if cfg == nil || !cfg.Enabled || cfg.TriggerType != autonomy.TriggerSchedule {
			continue
		}
		triggerCfg := *cfg
		runTrigger := func(runCtx context.Context) error {
			return executeDaemonTrigger(runCtx, cmd, app, &triggerCfg, "", &autonomy.TriggerEvent{
				TriggerID:   triggerCfg.TriggerID,
				TriggerType: triggerCfg.TriggerType,
				SourceEvent: "schedule.tick",
			})
		}
		if interval, ok := parseDaemonIntervalPattern(triggerCfg.Pattern); ok {
			automation.scheduler.Register(autonomy.SchedulerTask{
				Name:     "trigger:" + triggerCfg.TriggerID,
				Interval: interval,
				Action:   runTrigger,
			})
			continue
		}
		if schedule, ok := parseDaemonCronPattern(triggerCfg.Pattern); ok {
			if _, err := automation.cron.AddFunc(schedule, func() {
				_ = runTrigger(ctx)
			}); err != nil {
				return nil, fmt.Errorf("register trigger cron %s: %w", triggerCfg.TriggerID, err)
			}
		}
	}

	automation.scheduler.Start(ctx)
	automation.cron.Start()
	return automation, nil
}

func executeDaemonTrigger(ctx context.Context, cmd *cobra.Command, app bootstrap.App, cfg *autonomy.TriggerConfig, repoOverride string, ev *autonomy.TriggerEvent) error {
	if cfg == nil {
		return fmt.Errorf("trigger config is required")
	}
	repoRoot := firstNonEmpty(app.RepoRoot, app.Config.Paths.RepositoryRoot)
	owner, repoName := resolveTriggerRepoWithOverride(cfg, repoOverride, repoRoot)
	repoRoot = selectRepoRootForRemote(app, repoRoot, owner, repoName)

	result, err := runAutonomyCycle(ctx, cmd, app, autonomyRunRequest{
		RepoRoot:          repoRoot,
		Owner:             owner,
		Repo:              repoName,
		Intent:            firstNonEmpty(strings.TrimSpace(cfg.ActionTemplate), "respond to trigger"),
		Execute:           true,
		AutoThreshold:     autonomy.RiskLow,
		ApprovalThreshold: autonomy.RiskMedium,
	})
	if err != nil {
		return err
	}

	if ev != nil {
		ev.ResultingTaskID = result.Report.CycleID
		if strings.TrimSpace(ev.TriggerID) == "" {
			ev.TriggerID = cfg.TriggerID
		}
		if ev.TriggerType == "" {
			ev.TriggerType = cfg.TriggerType
		}
		if strings.TrimSpace(ev.SourceEvent) == "" {
			ev.SourceEvent = firstNonEmpty(strings.TrimSpace(cfg.Source), "daemon")
		}
		if err := app.StorageProvider.TriggerStore().AppendTriggerEvent(ev); err != nil {
			return err
		}
	}
	return nil
}

func resolveTriggerRepoWithOverride(cfg *autonomy.TriggerConfig, overrideRepo, repoRoot string) (string, string) {
	if owner, repoName := parseRepoFlag(strings.TrimSpace(overrideRepo), repoRoot); owner != "" && repoName != "" {
		return owner, repoName
	}
	return resolveTriggerRepo(cfg, "", repoRoot)
}

func parseDaemonIntervalPattern(raw string) (time.Duration, bool) {
	pattern := strings.TrimSpace(raw)
	if pattern == "" {
		return 0, false
	}
	if value, err := time.ParseDuration(pattern); err == nil && value > 0 {
		return value, true
	}
	if strings.HasPrefix(strings.ToLower(pattern), "@every ") {
		value, err := time.ParseDuration(strings.TrimSpace(pattern[len("@every "):]))
		if err == nil && value > 0 {
			return value, true
		}
	}
	return 0, false
}

func parseDaemonCronPattern(raw string) (string, bool) {
	pattern := strings.TrimSpace(raw)
	if pattern == "" {
		return "", false
	}
	if _, ok := parseDaemonIntervalPattern(pattern); ok {
		return "", false
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := parser.Parse(pattern); err != nil {
		return "", false
	}
	return pattern, true
}
