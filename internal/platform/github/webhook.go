package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

// WebhookHandler validates and parses GitHub webhook events.
type WebhookHandler struct {
	secret string
}

// NewWebhookHandler creates a new WebhookHandler with the given secret.
func NewWebhookHandler(secret string) *WebhookHandler {
	return &WebhookHandler{secret: secret}
}

// ValidateSignature verifies the X-Hub-Signature-256 header against the request body.
func (h *WebhookHandler) ValidateSignature(r *http.Request, body []byte) error {
	if h.secret == "" {
		return fmt.Errorf("webhook secret is not configured")
	}
	sig := r.Header.Get("X-Hub-Signature-256")
	if sig == "" {
		return fmt.Errorf("missing X-Hub-Signature-256 header")
	}
	if !strings.HasPrefix(sig, "sha256=") {
		return fmt.Errorf("invalid signature format: expected sha256=...")
	}
	sigHex := strings.TrimPrefix(sig, "sha256=")

	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sigHex), []byte(expected)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

// ParseEvent extracts the event type and payload from the request.
// The body should be the raw request body (typically read before calling ValidateSignature).
func (h *WebhookHandler) ParseEvent(r *http.Request, body []byte) (eventType string, payload []byte, err error) {
	eventType = r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		return "", nil, fmt.Errorf("missing X-GitHub-Event header")
	}
	payload = body
	return eventType, payload, nil
}
