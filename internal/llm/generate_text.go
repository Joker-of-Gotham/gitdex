package llm

import (
	"context"
	"fmt"
	"strings"
)

// GenerateText prefers streaming generation and aggregates chunks into one string.
// It falls back to non-streaming generation when streaming is unavailable.
func GenerateText(ctx context.Context, provider LLMProvider, req GenerateRequest) (*GenerateResponse, error) {
	if provider == nil {
		return nil, fmt.Errorf("llm provider not configured")
	}

	stream, err := provider.GenerateStream(ctx, req)
	if err == nil {
		var text strings.Builder
		chunkCount := 0

		for {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case chunk, ok := <-stream:
				if !ok {
					if chunkCount > 0 || text.Len() > 0 {
						return &GenerateResponse{
							Text:       strings.TrimSpace(text.String()),
							TokenCount: chunkCount,
						}, nil
					}
					goto fallback
				}
				text.WriteString(chunk.Text)
				chunkCount++
				if chunk.Done {
					return &GenerateResponse{
						Text:       strings.TrimSpace(text.String()),
						TokenCount: chunkCount,
					}, nil
				}
			}
		}
	}

fallback:
	return provider.Generate(ctx, req)
}
