package server

import "github.com/go-chi/chi/v5"

func (s *Server) registerRoutes() {
	r := s.router
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(securityHeaders)
		r.Use(jsonContentType)

		r.Route("/plans", func(r chi.Router) {
			r.Get("/", s.handleListPlans)
			r.Post("/", s.handleCreatePlan)
			r.Get("/{planID}", s.handleGetPlan)
		})

		r.Route("/tasks", func(r chi.Router) {
			r.Get("/", s.handleListTasks)
			r.Get("/{taskID}", s.handleGetTask)
		})

		r.Route("/audit", func(r chi.Router) {
			r.Get("/", s.handleQueryAudit)
		})

		r.Route("/campaigns", func(r chi.Router) {
			r.Get("/", s.handleListCampaigns)
			r.Post("/", s.handleCreateCampaign)
			r.Get("/{campaignID}", s.handleGetCampaign)
		})

		r.Route("/monitors", func(r chi.Router) {
			r.Get("/", s.handleListMonitors)
			r.Get("/{monitorID}", s.handleGetMonitor)
		})

		r.Get("/health", s.handleHealth)
		r.Post("/webhooks/github", s.handleGitHubWebhook)

		r.Route("/cruise", func(r chi.Router) {
			r.Get("/status", s.handleCruiseStatus)
			r.Post("/pause", s.handleCruisePause)
			r.Post("/resume", s.handleCruiseResume)
		})

		r.Get("/approvals", s.handleListApprovals)
		r.Post("/approvals/{id}/approve", s.handleApprovalApprove)
		r.Post("/approvals/{id}/reject", s.handleApprovalReject)

		r.Get("/metrics", s.handleMetrics)
	})
}
