package command

import "testing"

func TestBuildBootstrapOptionsOnlyMarksExplicitFlagsAsOverrides(t *testing.T) {
	got := buildBootstrapOptions(runtimeOptions{
		configFile: "configs/gitdex.example.yaml",
		output:     "text",
		logLevel:   "info",
		profile:    "local",
	}, "test-version", func(string) bool { return false })

	if got.ConfigFile != "configs/gitdex.example.yaml" {
		t.Fatalf("ConfigFile = %q, want %q", got.ConfigFile, "configs/gitdex.example.yaml")
	}

	if got.OutputSet {
		t.Fatal("OutputSet should be false when --output is not explicitly passed")
	}

	if got.LogLevelSet {
		t.Fatal("LogLevelSet should be false when --log-level is not explicitly passed")
	}

	if got.ProfileSet {
		t.Fatal("ProfileSet should be false when --profile is not explicitly passed")
	}
}

func TestBuildBootstrapOptionsMarksChangedFlags(t *testing.T) {
	changed := map[string]bool{
		"output":  true,
		"profile": true,
	}

	got := buildBootstrapOptions(runtimeOptions{
		output:   "json",
		logLevel: "info",
		profile:  "prod",
	}, "test-version", func(name string) bool {
		return changed[name]
	})

	if !got.OutputSet {
		t.Fatal("OutputSet should be true when --output is explicitly passed")
	}

	if got.LogLevelSet {
		t.Fatal("LogLevelSet should be false when --log-level is not explicitly passed")
	}

	if !got.ProfileSet {
		t.Fatal("ProfileSet should be true when --profile is explicitly passed")
	}

	if got.Output != "json" {
		t.Fatalf("Output = %q, want %q", got.Output, "json")
	}

	if got.Profile != "prod" {
		t.Fatalf("Profile = %q, want %q", got.Profile, "prod")
	}
}
