package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/tiktoken-go/tokenizer"
)

// openaiProvider counts tokens using a layered strategy:
//  1. tiktoken-go's ForModel (covers gpt-4o, gpt-4.1, gpt-5{,-mini,-nano},
//     o1/o3/o4-mini, gpt-3.5-turbo, etc. — both directly and via the
//     library's own internal prefix table).
//  2. Local prefix-matching fallback for newer model names that tiktoken-go
//     does not yet recognize (gpt-5.1, gpt-5.2, gpt-5.4, gpt-5.5, gpt-5-pro,
//     ...). All current OpenAI text models use either o200k_base
//     (gpt-4o / gpt-4.1 / gpt-5 / o-series) or cl100k_base (gpt-4 / gpt-3.5)
//     as their BPE encoding, so a prefix lookup is correct in practice and
//     avoids any network call.
//  3. Remote fallback to POST /v1/responses/input_tokens, which requires
//     OPENAI_API_KEY. With (1)+(2) covering every model in the catalog as
//     of 2026-05, this path is rarely reached.
type openaiProvider struct{}

func (p *openaiProvider) Name() string { return "openai" }

// prefixEncodingFor returns the tiktoken encoding implied by a model name's
// prefix, or "" if no rule matches. The order matters: more specific prefixes
// (gpt-3.5*, gpt-4o*, gpt-4.1*) must precede broader ones (gpt-4*) because
// HasPrefix is greedy from the top.
func prefixEncodingFor(model string) tokenizer.Encoding {
	switch {
	// o200k_base family — the modern BPE used by all gpt-4o / gpt-4.1 /
	// gpt-5 generations and by every reasoning (o-series) model.
	case strings.HasPrefix(model, "gpt-5"),
		strings.HasPrefix(model, "gpt-4o"),
		strings.HasPrefix(model, "gpt-4.1"),
		strings.HasPrefix(model, "gpt-4-vision"),
		strings.HasPrefix(model, "o1"),
		strings.HasPrefix(model, "o3"),
		strings.HasPrefix(model, "o4"):
		return tokenizer.O200kBase
	// cl100k_base family — gpt-3.5-turbo and base gpt-4 share this older
	// BPE. The gpt-4o / gpt-4.1 cases above must already have matched, so
	// "gpt-4" here genuinely means non-4o/4.1 gpt-4 variants.
	case strings.HasPrefix(model, "gpt-3.5"),
		strings.HasPrefix(model, "gpt-4"):
		return tokenizer.Cl100kBase
	}
	return ""
}

func (p *openaiProvider) Count(ctx context.Context, model string, text string) (int, error) {
	// 1. Attempt tiktoken-go's own model resolution.
	if enc, err := tokenizer.ForModel(tokenizer.Model(model)); err == nil {
		n, cErr := enc.Count(text)
		if cErr != nil {
			return 0, fmt.Errorf("openai: count failed: %w", cErr)
		}
		return n, nil
	}

	// 2. Local prefix-matching fallback. All current OpenAI text models map
	//    cleanly to either o200k_base or cl100k_base; matching by family
	//    prefix lets newly released models (gpt-5.5, gpt-5.4-pro, ...) be
	//    counted without any network call.
	if encName := prefixEncodingFor(model); encName != "" {
		enc, err := tokenizer.Get(encName)
		if err != nil {
			return 0, fmt.Errorf("openai: failed to load encoding %s: %w", encName, err)
		}
		n, cErr := enc.Count(text)
		if cErr != nil {
			return 0, fmt.Errorf("openai: count failed: %w", cErr)
		}
		return n, nil
	}

	// 3. Remote fallback for genuinely unrecognized models. Requires API key.
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return 0, fmt.Errorf("OPENAI_API_KEY environment variable is required for openai model %q (not in local tokenizer)", model)
	}

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
