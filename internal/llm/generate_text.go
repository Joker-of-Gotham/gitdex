package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/observability"
)

// GenerateText prefers streaming generation and aggregates chunks into one string.
// It falls back to non-streaming generation when streaming is unavailable.
// Fatal errors (model not found, auth failure) are returned immediately.
func GenerateText(ctx context.Context, provider LLMProvider, req GenerateRequest) (*GenerateResponse, error) {
	start := time.Now()
	success := false
	defer func() {
		observability.RecordLLMCall(time.Since(start), success)
	}()

	if provider == nil {
		return nil, fmt.Errorf("llm provider not configured")
	}

	stream, streamErr := provider.GenerateStream(ctx, req)
	if streamErr != nil {
		if isFatalProviderError(streamErr) {
			return nil, streamErr
		}
		resp, err := provider.Generate(ctx, req)
		success = err == nil
		return resp, err
	}

	var text strings.Builder
	var thinking strings.Builder
	chunkCount := 0

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case chunk, ok := <-stream:
			if !ok {
				if chunkCount > 0 || text.Len() > 0 || thinking.Len() > 0 {
					success = true
					return &GenerateResponse{
						Text:       strings.TrimSpace(text.String()),
						Thinking:   strings.TrimSpace(thinking.String()),
						TokenCount: chunkCount,
					}, nil
				}
				resp, err := provider.Generate(ctx, req)
				success = err == nil
				return resp, err
			}
			text.WriteString(chunk.Text)
			thinking.WriteString(chunk.Thinking)
			chunkCount++
			if chunk.Done {
				success = true
				return &GenerateResponse{
					Text:       strings.TrimSpace(text.String()),
					Thinking:   strings.TrimSpace(thinking.String()),
					TokenCount: chunkCount,
				}, nil
			}
		}
	}
}

func isFatalProviderError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "404") ||
		strings.Contains(msg, "401") ||
		strings.Contains(msg, "403") ||
		strings.Contains(msg, "missing API key")
}
