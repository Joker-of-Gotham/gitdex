package tui

import (
	"fmt"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
	platformruntime "github.com/Joker-of-Gotham/gitdex/internal/platform/runtime"
)

type platformPreflightResult struct {
	Platform    platform.Platform
	Request     platformExecRequest
	Diagnostics platform.DiagnosticSet
	Bundle      *platformruntime.Bundle
}

func (m Model) preflightPlatformRequest(req platformExecRequest) (platformPreflightResult, error) {
	result := platformPreflightResult{Request: req}
	if req.Op == nil {
		return result, fmt.Errorf("platform request is missing operation metadata")
	}

	platformID := m.detectedPlatform()
	if platformID == platform.PlatformUnknown && m.gitState != nil {
		platformID = platform.DetectPlatform(platform.PreferredRemoteURL(m.gitState.RemoteInfos))
	}
	result.Platform = platformID

	diagnostics, repaired := platform.DiagnosePlatformOperation(platformID, m.gitState, clonePlatformExecInfo(req.Op))
	if repaired != nil {
		req.Op = repaired
		result.Request = req
	}
	result.Diagnostics = diagnostics
	if diagnostics.Decision == platform.DiagnosticBlocked {
		return result, fmt.Errorf("diagnostic blocked execution: %s", summarizeDiagnostics(diagnostics))
	}

	resolve := m.resolveAdminBundle
	if resolve == nil {
		resolve = platformruntime.ResolveAdminBundle
	}
	bundle, err := resolve(m.gitState, m.platformCfg, m.adapterCfg)
	if err != nil {
		return result, err
	}
	exec := bundle.Executors[strings.TrimSpace(req.Op.CapabilityID)]
	if exec == nil {
		return result, fmt.Errorf("platform executor %q is unavailable on %s", req.Op.CapabilityID, bundle.Platform.String())
	}
	adapterExec := bundle.ExecutorAdapter
	if adapterExec == nil {
		adapterExec = platform.NewDirectAdapterExecutor(bundle.Adapter)
	}
	if !adapterExec.CanHandle(strings.TrimSpace(req.Op.CapabilityID)) {
		return result, fmt.Errorf("%s adapter cannot handle capability %q", adapterExec.Kind(), req.Op.CapabilityID)
	}

	result.Platform = bundle.Platform
	result.Bundle = bundle
	return result, nil
}

