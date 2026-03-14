package config

import "testing"

func TestApplyAutomationModeDerivesFlags(t *testing.T) {
	cfg := AutomationConfig{Mode: AutomationModeCruise, MonitorInterval: 120}
	ApplyAutomationMode(&cfg)

	if !cfg.Enabled || !cfg.AutoAnalyze || !cfg.Unattended || !cfg.AutoAcceptSafe {
		t.Fatalf("cruise mode did not enable the expected automation flags: %+v", cfg)
	}
}

func TestAutomationModeFromFlagsBackfillsLegacyConfigs(t *testing.T) {
	cfg := AutomationConfig{
		Enabled:        true,
		AutoAnalyze:    true,
		Unattended:     true,
		AutoAcceptSafe: true,
	}

	if got := AutomationModeFromFlags(cfg); got != AutomationModeAuto {
		t.Fatalf("expected %q, got %q", AutomationModeAuto, got)
	}
}
