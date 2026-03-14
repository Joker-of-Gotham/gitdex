package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeAutomationMode_AllInputs(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"manual", AutomationModeManual},
		{"Manual", AutomationModeManual},
		{" MANUAL ", AutomationModeManual},
		{"assist", AutomationModeManual},
		{"Assist", AutomationModeManual},
		{"auto", AutomationModeAuto},
		{"AUTO", AutomationModeAuto},
		{"cruise", AutomationModeCruise},
		{"CRUISE", AutomationModeCruise},
		{"", AutomationModeManual},
		{"invalid", AutomationModeManual},
		{"semi-auto", AutomationModeManual},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeAutomationMode(tt.input))
		})
	}
}

func TestAutomationModeIsAutoLoop_Classification(t *testing.T) {
	assert.False(t, AutomationModeIsAutoLoop("manual"))
	assert.False(t, AutomationModeIsAutoLoop("assist"))
	assert.False(t, AutomationModeIsAutoLoop(""))
	assert.False(t, AutomationModeIsAutoLoop("unknown"))
	assert.True(t, AutomationModeIsAutoLoop("auto"))
	assert.True(t, AutomationModeIsAutoLoop("AUTO"))
	assert.True(t, AutomationModeIsAutoLoop("cruise"))
	assert.True(t, AutomationModeIsAutoLoop("CRUISE"))
}

func TestApplyAutomationMode_ManualDisablesUnattended(t *testing.T) {
	cfg := AutomationConfig{Mode: "manual"}
	ApplyAutomationMode(&cfg)
	assert.True(t, cfg.Enabled)
	assert.True(t, cfg.AutoAnalyze)
	assert.False(t, cfg.Unattended)
	assert.False(t, cfg.AutoAcceptSafe)
}

func TestApplyAutomationMode_AutoEnablesUnattended(t *testing.T) {
	cfg := AutomationConfig{Mode: "auto"}
	ApplyAutomationMode(&cfg)
	assert.True(t, cfg.Enabled)
	assert.True(t, cfg.AutoAnalyze)
	assert.True(t, cfg.Unattended)
	assert.True(t, cfg.AutoAcceptSafe)
}

func TestApplyAutomationMode_CruiseEnablesAll(t *testing.T) {
	cfg := AutomationConfig{Mode: "cruise"}
	ApplyAutomationMode(&cfg)
	assert.True(t, cfg.Enabled)
	assert.True(t, cfg.AutoAnalyze)
	assert.True(t, cfg.Unattended)
	assert.True(t, cfg.AutoAcceptSafe)
}

func TestApplyAutomationMode_NilSafe(t *testing.T) {
	ApplyAutomationMode(nil)
}

func TestAutomationModeAllowsSelfDirectedGoals_OnlyCruise(t *testing.T) {
	assert.False(t, AutomationModeAllowsSelfDirectedGoals("manual"))
	assert.False(t, AutomationModeAllowsSelfDirectedGoals("auto"))
	assert.True(t, AutomationModeAllowsSelfDirectedGoals("cruise"))
}

func TestApplyAutomationMode_AssistNormalizesToManual(t *testing.T) {
	cfg := AutomationConfig{Mode: "assist"}
	ApplyAutomationMode(&cfg)
	assert.Equal(t, AutomationModeManual, cfg.Mode)
	assert.False(t, cfg.Unattended)
	assert.False(t, cfg.AutoAcceptSafe)
}
