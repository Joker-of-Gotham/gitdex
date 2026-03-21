package autonomy

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type CruiseState string

const (
	CruiseIdle    CruiseState = "idle"
	CruiseRunning CruiseState = "running"
	CruisePaused  CruiseState = "paused"
)

type RetryPolicy struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

type DeadLetterEntry struct {
	Action    string
	Error     string
	Timestamp time.Time
	Retries   int
}

// MetricsSummary is a dashboard-friendly snapshot for HTTP/TUI surfaces.
type MetricsSummary struct {
	CycleCount           int     `json:"cycle_count"`
	SuccessRate          float64 `json:"success_rate"`
	MeanCycleDurationMs  int64   `json:"mean_cycle_duration_ms"`
	SuccessfulExecutions int64   `json:"successful_executions"`
	TotalExecutions      int64   `json:"total_executions"`
}

// DeadLetterSummary exposes dead-letter queue entries for API consumers.
type DeadLetterSummary struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Reason      string `json:"reason"`
}

// PendingApproval is a plan waiting for explicit human approval.
type PendingApproval struct {
	ID     string `json:"id"`
	PlanID string `json:"plan_id"`
	Title  string `json:"title,omitempty"`
}

type CruiseMetrics struct {
	CycleCount     int64
	SuccessCount   int64
	FailureCount   int64
	MeanDurationMs float64
	LastCycleAt    time.Time
}

type CruiseConfig struct {
	Enabled              bool          `json:"enabled" yaml:"enabled"`
	Interval             time.Duration `json:"interval" yaml:"interval"`
	AutoExecuteThreshold RiskLevel     `json:"auto_execute_threshold" yaml:"auto_execute_threshold"`
	ApprovalThreshold    RiskLevel     `json:"approval_threshold" yaml:"approval_threshold"`
	IdempotencyWindow    time.Duration `json:"idempotency_window" yaml:"idempotency_window"`
	RetryPolicy          RetryPolicy   `json:"retry_policy" yaml:"retry_policy"`
}

func DefaultCruiseConfig() CruiseConfig {
	return CruiseConfig{
		Enabled:              false,
		Interval:             30 * time.Minute,
		AutoExecuteThreshold: RiskLow,
		ApprovalThreshold:    RiskMedium,
		IdempotencyWindow:    time.Hour,
		RetryPolicy: RetryPolicy{
			MaxRetries:     3,
			InitialBackoff: time.Second,
			MaxBackoff:     30 * time.Second,
		},
	}
}

type CruiseEngine struct {
	mu       sync.Mutex
	state    CruiseState
	config   CruiseConfig
	planner  *Planner
	guard    *Guardrails
	executor *PlanExecutor
	reporter *Reporter

	policyGate *PolicyGate

	repoLocks   map[string]*sync.Mutex
	locksMu     sync.RWMutex
	retryPolicy RetryPolicy
	deadLetter  []DeadLetterEntry
	dlMu        sync.Mutex
	idempotency map[string]time.Time
	idMu        sync.RWMutex
	metrics     CruiseMetrics
	metricsMu   sync.RWMutex

	cancel     context.CancelFunc
	lastCycle  *CruiseReport
	cycleCount int
	onReport   func(CruiseReport)

	pendingApprovals []PendingApproval
	pendingMu        sync.Mutex
}

type CruiseOption func(*CruiseEngine)

func WithPolicyGate(pg *PolicyGate) CruiseOption {
	return func(e *CruiseEngine) {
		e.policyGate = pg
	}
}

func WithRetryPolicy(r RetryPolicy) CruiseOption {
	return func(e *CruiseEngine) {
		e.retryPolicy = r
	}
}

func NewCruiseEngine(cfg CruiseConfig, planner *Planner, guard *Guardrails, executor *PlanExecutor, reporter *Reporter, opts ...CruiseOption) *CruiseEngine {
	e := &CruiseEngine{
		state:       CruiseIdle,
		config:      cfg,
		planner:     planner,
		guard:       guard,
		executor:    executor,
		reporter:    reporter,
		policyGate:  NewPolicyGate(),
		repoLocks:   make(map[string]*sync.Mutex),
		retryPolicy: cfg.RetryPolicy,
		idempotency: make(map[string]time.Time),
	}
	if e.retryPolicy.MaxRetries <= 0 && e.retryPolicy.InitialBackoff == 0 && e.retryPolicy.MaxBackoff == 0 {
		e.retryPolicy = RetryPolicy{
			MaxRetries:     3,
			InitialBackoff: time.Second,
			MaxBackoff:     30 * time.Second,
		}
	}
	for _, o := range opts {
		o(e)
	}
	if executor != nil {
		executor.SetPolicyGate(e.policyGate)
	}
	return e
}

func (e *CruiseEngine) SetReportHandler(fn func(CruiseReport)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onReport = fn
}

func (e *CruiseEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.state == CruiseRunning {
		e.mu.Unlock()
		return fmt.Errorf("cruise already running")
	}
	e.state = CruiseRunning
	runCtx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	e.mu.Unlock()

	go e.loop(runCtx)
	return nil
}

func (e *CruiseEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cancel != nil {
		e.cancel()
		e.cancel = nil
	}
	e.state = CruiseIdle
}

func (e *CruiseEngine) Pause() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state == CruiseRunning {
		e.state = CruisePaused
	}
}

func (e *CruiseEngine) Resume() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state == CruisePaused {
		e.state = CruiseRunning
	}
}

func (e *CruiseEngine) State() CruiseState {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state
}

func (e *CruiseEngine) LastReport() *CruiseReport {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.lastCycle
}

func (e *CruiseEngine) CycleCount() int {
	e.metricsMu.RLock()
	defer e.metricsMu.RUnlock()
	return int(e.metrics.CycleCount)
}

func (e *CruiseEngine) Metrics() CruiseMetrics {
	e.metricsMu.RLock()
	defer e.metricsMu.RUnlock()
	return e.metrics
}

func (e *CruiseEngine) DeadLetterEntries() []DeadLetterEntry {
	e.dlMu.Lock()
	defer e.dlMu.Unlock()
	out := make([]DeadLetterEntry, len(e.deadLetter))
	copy(out, e.deadLetter)
	return out
}

func (e *CruiseEngine) MetricsSummary() MetricsSummary {
	e.metricsMu.RLock()
	m := e.metrics
	e.metricsMu.RUnlock()
	total := m.SuccessCount + m.FailureCount
	var rate float64
	if total > 0 {
		rate = float64(m.SuccessCount) / float64(total)
	}
	return MetricsSummary{
		CycleCount:           int(m.CycleCount),
		SuccessRate:          rate,
		MeanCycleDurationMs:  int64(m.MeanDurationMs),
		SuccessfulExecutions: m.SuccessCount,
		TotalExecutions:      total,
	}
}

func (e *CruiseEngine) DeadLetterSummaries() []DeadLetterSummary {
	e.dlMu.Lock()
	defer e.dlMu.Unlock()
	out := make([]DeadLetterSummary, 0, len(e.deadLetter))
	for i, d := range e.deadLetter {
		out = append(out, DeadLetterSummary{
			ID:          fmt.Sprintf("dl-%d-%d", d.Timestamp.Unix(), i),
			Kind:        "execution",
			Description: d.Action,
			Reason:      d.Error,
		})
	}
	return out
}

func (e *CruiseEngine) PendingApprovals() []PendingApproval {
	e.pendingMu.Lock()
	defer e.pendingMu.Unlock()
	out := make([]PendingApproval, len(e.pendingApprovals))
	copy(out, e.pendingApprovals)
	return out
}

func (e *CruiseEngine) Approve(_ context.Context, id string) error {
	e.pendingMu.Lock()
	defer e.pendingMu.Unlock()
	for i, p := range e.pendingApprovals {
		if p.ID == id {
			e.pendingApprovals = append(e.pendingApprovals[:i], e.pendingApprovals[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("approval %q not found", id)
}

func (e *CruiseEngine) Reject(_ context.Context, id string) error {
	e.pendingMu.Lock()
	defer e.pendingMu.Unlock()
	for i, p := range e.pendingApprovals {
		if p.ID == id {
			e.pendingApprovals = append(e.pendingApprovals[:i], e.pendingApprovals[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("approval %q not found", id)
}

func (e *CruiseEngine) Reporter() *Reporter {
	return e.reporter
}

func (e *CruiseEngine) loop(ctx context.Context) {
	ticker := time.NewTicker(e.config.Interval)
	defer ticker.Stop()

	e.runCycle(ctx)

	for {
		select {
		case <-ctx.Done():
			e.mu.Lock()
			e.state = CruiseIdle
			e.mu.Unlock()
			return
		case <-ticker.C:
			e.mu.Lock()
			st := e.state
			e.mu.Unlock()
			if st == CruiseRunning {
				e.runCycle(ctx)
			}
		}
	}
}

func (e *CruiseEngine) runCycle(ctx context.Context) {
	startTime := time.Now()
	e.mu.Lock()
	e.cycleCount++
	cycleID := fmt.Sprintf("cycle-%d", e.cycleCount)
	e.mu.Unlock()

	report := CruiseReport{
		CycleID:   cycleID,
		StartTime: startTime,
	}

	cycleOK := true

	plans, err := e.planner.AnalyzeAndPlan(ctx)
	if err != nil {
		cycleOK = false
		report.Errors = append(report.Errors, fmt.Sprintf("planner error: %v", err))
	}

	for _, plan := range plans {
		repoKey := repoKeyForPlan(plan)
		unlock := e.acquireRepoLock(repoKey)
		func() {
			defer unlock()

			planRisk := e.guard.EvaluateRisk(plan)
			primary := primaryAction(plan)

			if e.policyGate != nil {
				gate, reason := e.policyGate.Evaluate(primary, planRisk)
				switch gate {
				case GateBlocked:
					report.Blocked = append(report.Blocked, BlockedAction{
						Plan:   plan,
						Reason: reason,
					})
					cycleOK = false
					return
				case GateManual:
					report.Pending = append(report.Pending, plan)
					return
				}
				if need, n := e.policyGate.RequiresApproval(primary); need && n > 0 {
					report.Pending = append(report.Pending, plan)
					return
				}
				if e.policyGate.HasMissionWindows() && !e.policyGate.IsInMissionWindow() {
					report.Pending = append(report.Pending, plan)
					return
				}
			} else {
				if planRisk <= e.config.AutoExecuteThreshold {
					// proceed to guard + execute
				} else if planRisk <= e.config.ApprovalThreshold {
					report.Pending = append(report.Pending, plan)
					return
				} else {
					allowed, reason := e.guard.CheckPolicy(plan)
					if !allowed {
						report.Blocked = append(report.Blocked, BlockedAction{
							Plan:   plan,
							Reason: reason,
						})
						cycleOK = false
					} else {
						report.Pending = append(report.Pending, plan)
					}
					return
				}
			}

			allowed, reason := e.guard.CheckPolicy(plan)
			if !allowed {
				report.Blocked = append(report.Blocked, BlockedAction{
					Plan:   plan,
					Reason: reason,
				})
				cycleOK = false
				return
			}

			idKey := idempotencyKey(plan, primary)
			if e.shouldSkipIdempotent(idKey) {
				report.Errors = append(report.Errors, fmt.Sprintf("idempotent skip: %s", idKey))
				return
			}

			result := e.executeWithRetries(ctx, plan, primary)
			report.Executed = append(report.Executed, ExecutedAction{
				Plan:   plan,
				Result: result,
			})
			if !result.Success {
				cycleOK = false
			} else {
				e.markIdempotent(idKey)
			}
		}()
	}

	report.EndTime = time.Now()
	e.recordCycleMetrics(cycleOK, time.Since(startTime))

	e.mu.Lock()
	e.lastCycle = &report
	handler := e.onReport
	e.mu.Unlock()

	if e.reporter != nil {
		e.reporter.Add(report)
	}

	if handler != nil {
		handler(report)
	}
}

func primaryAction(plan ActionPlan) string {
	if len(plan.Steps) == 0 {
		if plan.ID != "" {
			return plan.ID
		}
		return "plan"
	}
	return plan.Steps[0].Action
}

func repoKeyForPlan(plan ActionPlan) string {
	if plan.ID != "" {
		return plan.ID
	}
	for _, s := range plan.Steps {
		if p := s.Args["path"]; p != "" {
			return p
		}
		if r := s.Args["repo"]; r != "" {
			return r
		}
	}
	return "default"
}

func idempotencyKey(plan ActionPlan, primary string) string {
	pid := plan.ID
	if pid == "" {
		pid = plan.Description
	}
	return fmt.Sprintf("%s:%s", pid, primary)
}

func (e *CruiseEngine) acquireRepoLock(repoKey string) func() {
	e.locksMu.Lock()
	m, ok := e.repoLocks[repoKey]
	if !ok {
		m = &sync.Mutex{}
		e.repoLocks[repoKey] = m
	}
	e.locksMu.Unlock()
	m.Lock()
	return m.Unlock
}

func (e *CruiseEngine) shouldSkipIdempotent(key string) bool {
	window := e.config.IdempotencyWindow
	if window <= 0 {
		window = time.Hour
	}
	e.idMu.RLock()
	defer e.idMu.RUnlock()
	last, ok := e.idempotency[key]
	if !ok {
		return false
	}
	return time.Since(last) < window
}

func (e *CruiseEngine) markIdempotent(key string) {
	now := time.Now()
	e.idMu.Lock()
	defer e.idMu.Unlock()
	e.idempotency[key] = now
}

func (e *CruiseEngine) executeWithRetries(ctx context.Context, plan ActionPlan, primary string) ExecutionResult {
	var last ExecutionResult
	if e.executor == nil {
		last.Error = "no executor configured"
		return last
	}
	attempts := e.retryPolicy.MaxRetries
	if attempts < 0 {
		attempts = 0
	}
	maxAttempts := attempts + 1

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			backoff := e.retryPolicy.InitialBackoff * time.Duration(1<<uint(attempt-1))
			if e.retryPolicy.MaxBackoff > 0 && backoff > e.retryPolicy.MaxBackoff {
				backoff = e.retryPolicy.MaxBackoff
			}
			select {
			case <-ctx.Done():
				last.Error = ctx.Err().Error()
				return last
			case <-time.After(backoff):
			}
		}

		last = e.executor.Execute(ctx, plan)
		if last.Success {
			return last
		}
	}

	e.dlMu.Lock()
	e.deadLetter = append(e.deadLetter, DeadLetterEntry{
		Action:    primary,
		Error:     last.Error,
		Timestamp: time.Now().UTC(),
		Retries:   attempts,
	})
	e.dlMu.Unlock()

	return last
}

func (e *CruiseEngine) recordCycleMetrics(success bool, d time.Duration) {
	e.metricsMu.Lock()
	defer e.metricsMu.Unlock()
	e.metrics.CycleCount++
	if success {
		e.metrics.SuccessCount++
	} else {
		e.metrics.FailureCount++
	}
	n := float64(e.metrics.CycleCount)
	e.metrics.MeanDurationMs = (e.metrics.MeanDurationMs*(n-1) + float64(d.Milliseconds())) / n
	e.metrics.LastCycleAt = time.Now().UTC()
}
