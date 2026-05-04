package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/tiktoken-go/tokenizer"
)

// openaiProvider counts tokens using a hybrid strategy:
//  1. Try local tokenizer via tiktoken-go (no network/key required for known models).
//  2. On unknown model, fall back to the OpenAI remote tokenization endpoint.
//     Remote fallback requires OPENAI_API_KEY.
type openaiProvider struct{}

func (p *openaiProvider) Name() string { return "openai" }

func (p *openaiProvider) Count(ctx context.Context, model string, text string) (int, error) {
	// 1. Attempt local tokenization via tiktoken-go.
	enc, err := tokenizer.ForModel(tokenizer.Model(model))
	if err == nil {
		n, err := enc.Count(text)
		if err != nil {
			return 0, fmt.Errorf("openai: count failed: %w", err)
		}
		return n, nil
	}
	// tiktoken-go does not know this model — fall through to remote.

	// 2. Remote fallback: require API key.
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return 0, fmt.Errorf("OPENAI_API_KEY environment variable is required for openai model %q (not in local tokenizer)", model)
	}

	// Build request body.
	reqBody, err := json.Marshal(map[string]string{
		"model": model,
		"input": text,
	})
	if err != nil {
		return 0, fmt.Errorf("openai: failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openai.com/v1/responses/input_tokens",
		bytes.NewReader(reqBody))
	if err != nil {
		return 0, fmt.Errorf("openai: failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("openai: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("openai: failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("openai: request failed: %s %s", resp.Status, string(body))
	}

	var result struct {
		InputTokens int `json:"input_tokens"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("openai: failed to parse response: %w", err)
	}

	return result.InputTokens, nil
}
