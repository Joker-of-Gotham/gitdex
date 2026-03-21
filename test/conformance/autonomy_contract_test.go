package conformance

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/autonomy"
)

func TestAutonomyConfigContract_JSONFieldNames_SnakeCase(t *testing.T) {
	cfg := &autonomy.AutonomyConfig{
		ConfigID:             "cfg_contract",
		Name:                 "test",
		DefaultLevel:         autonomy.LevelManual,
		CapabilityAutonomies: nil,
		CreatedAt:            time.Now().UTC(),
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	jsonStr := string(data)
	requiredFields := []string{
		"config_id", "name", "capability_autonomies", "default_level", "created_at",
	}

	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, "\""+field+"\"") {
			t.Errorf("JSON missing snake_case field %q", field)
		}
	}
}

func TestCapabilityAutonomyContract_SnakeCase(t *testing.T) {
	ca := autonomy.CapabilityAutonomy{
		Capability:       "repo_sync",
		Level:            autonomy.LevelSupervised,
		Constraints:      []string{"branch=main"},
		RequiresApproval: true,
	}

	data, err := json.Marshal(ca)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	jsonStr := string(data)
	requiredFields := []string{"capability", "level", "constraints", "requires_approval"}
	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, "\""+field+"\"") {
			t.Errorf("CapabilityAutonomy JSON missing snake_case field %q", field)
		}
	}
}

func TestAutonomyLevelValues(t *testing.T) {
	levels := []autonomy.AutonomyLevel{
		autonomy.LevelManual,
		autonomy.LevelSupervised,
		autonomy.LevelAutonomous,
		autonomy.LevelFullAuto,
	}

	for _, l := range levels {
		if strings.ToLower(string(l)) != string(l) {
			t.Errorf("autonomy level %q should be lowercase", l)
		}
	}
}
