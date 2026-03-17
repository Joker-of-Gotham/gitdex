package config

const CurrentConfigVersion = 3

// MigrationInfo records applied config migrations.
type MigrationInfo struct {
	FromVersion int      `json:"from_version"`
	ToVersion   int      `json:"to_version"`
	Steps       []string `json:"steps"`
}

// Migrate applies in-memory config migrations to the latest version.
// This pipeline is intentionally additive and backward-compatible.
func Migrate(c *Config) MigrationInfo {
	info := MigrationInfo{
		FromVersion: c.Version,
		ToVersion:   CurrentConfigVersion,
	}
	if c == nil {
		return info
	}

	if c.Version <= 0 {
		info.Steps = append(info.Steps, "version missing -> default to v3")
		c.Version = CurrentConfigVersion
		return info
	}

	if c.Version < 2 {
		info.Steps = append(info.Steps, "legacy llm fields synchronized to role-based primary/secondary")
		if c.LLM.Primary.Model == "" {
			c.LLM.Primary.Model = c.LLM.Model
		}
		if c.LLM.Primary.Provider == "" {
			c.LLM.Primary.Provider = c.LLM.Provider
		}
		if c.LLM.Primary.Endpoint == "" {
			c.LLM.Primary.Endpoint = c.LLM.Endpoint
		}
	}

	if c.Version < 3 {
		info.Steps = append(info.Steps, "automation mode normalized and defaults hardened")
		ApplyAutomationMode(&c.Automation)
		if c.Automation.MonitorInterval <= 0 {
			c.Automation.MonitorInterval = 900
		}
	}

	c.Version = CurrentConfigVersion
	return info
}

