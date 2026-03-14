package config

import "strings"

const (
	AutomationModeManual = "manual"
	AutomationModeAssist = "manual" // legacy alias
	AutomationModeAuto   = "auto"
	AutomationModeCruise = "cruise"
)

func NormalizeAutomationMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "manual":
		return AutomationModeManual
	case "assist":
		return AutomationModeManual
	case "auto":
		return AutomationModeAuto
	case "cruise":
		return AutomationModeCruise
	default:
		return AutomationModeManual
	}
}

func AutomationModeFromFlags(cfg AutomationConfig) string {
	switch {
	case cfg.AutoAnalyze && cfg.Unattended && cfg.AutoAcceptSafe:
		return AutomationModeAuto
	default:
		return AutomationModeManual
	}
}

func ApplyAutomationMode(cfg *AutomationConfig) {
	if cfg == nil {
		return
	}
	cfg.Mode = NormalizeAutomationMode(firstNonEmpty(cfg.Mode, AutomationModeFromFlags(*cfg)))
	switch cfg.Mode {
	case AutomationModeManual:
		cfg.Enabled = true
		cfg.AutoAnalyze = true
		cfg.Unattended = false
		cfg.AutoAcceptSafe = false
	case AutomationModeAuto:
		cfg.Enabled = true
		cfg.AutoAnalyze = true
		cfg.Unattended = true
		cfg.AutoAcceptSafe = true
	case AutomationModeCruise:
		cfg.Enabled = true
		cfg.AutoAnalyze = true
		cfg.Unattended = true
		cfg.AutoAcceptSafe = true
	}
}

func AutomationModeAllowsSelfDirectedGoals(mode string) bool {
	return NormalizeAutomationMode(mode) == AutomationModeCruise
}

func AutomationModeIsAutoLoop(mode string) bool {
	m := NormalizeAutomationMode(mode)
	return m == AutomationModeAuto || m == AutomationModeCruise
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
