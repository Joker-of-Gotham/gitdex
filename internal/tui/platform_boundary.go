package tui

import (
	"fmt"
	"strings"

	gitplatform "github.com/Joker-of-Gotham/gitdex/internal/platform"
)

func (m Model) detectedPlatform() gitplatform.Platform {
	if m.cachedPlatformID != gitplatform.PlatformUnknown {
		return m.cachedPlatformID
	}
	return m.detectPlatformFromState()
}

func (m Model) detectPlatformFromState() gitplatform.Platform {
	if m.gitState != nil {
		if platformID := detectWorkflowPlatform(m.gitState); platformID != gitplatform.PlatformUnknown {
			return platformID
		}
	}
	if m.analysisTrace.PlatformState != nil {
		if platformID := gitplatform.ParsePlatform(m.analysisTrace.PlatformState.Detected); platformID != gitplatform.PlatformUnknown {
			return platformID
		}
	}
	return gitplatform.PlatformUnknown
}

func (m *Model) refreshCachedPlatform() {
	m.cachedPlatformID = m.detectPlatformFromState()
}

func (m Model) capabilityBoundary(capabilityID string) (gitplatform.CapabilityBoundary, bool) {
	platformID := m.detectedPlatform()
	if platformID == gitplatform.PlatformUnknown {
		return gitplatform.CapabilityBoundary{}, false
	}
	return gitplatform.CapabilityBoundaryFor(platformID, strings.TrimSpace(capabilityID))
}

func (m Model) capabilityCoverageLabel(capabilityID string) string {
	if boundary, ok := m.capabilityBoundary(capabilityID); ok {
		return boundary.Mode
	}
	if strings.TrimSpace(capabilityID) == "" {
		return ""
	}
	return "full"
}

func (m Model) capabilityCoverageSummary(capabilityIDs []string) string {
	if len(capabilityIDs) == 0 {
		return ""
	}
	counts := map[string]int{}
	for _, capabilityID := range capabilityIDs {
		mode := m.capabilityCoverageLabel(capabilityID)
		if mode == "" {
			continue
		}
		counts[mode]++
	}
	if len(counts) == 0 {
		return ""
	}
	order := []string{"full", "partial_mutate", "inspect_only", "composed"}
	parts := make([]string, 0, len(order))
	for _, mode := range order {
		if counts[mode] > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", mode, counts[mode]))
		}
	}
	return strings.Join(parts, " | ")
}

func gitPlatformBoundary(platformID gitplatform.Platform, capabilityID string) (gitplatform.CapabilityBoundary, bool) {
	if platformID == gitplatform.PlatformUnknown {
		platformID = gitplatform.PlatformGitHub
	}
	return gitplatform.CapabilityBoundaryFor(platformID, strings.TrimSpace(capabilityID))
}
