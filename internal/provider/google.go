package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	"google.golang.org/genai"
	"google.golang.org/genai/tokenizer"
)

// newLocalTokenizerSilent calls tokenizer.NewLocalTokenizer while suppressing
// the SDK's experimental warning that is unconditionally printed to os.Stdout.
// The warning fires via sync.Once on the first call, before any model-name
// validation, so we must redirect stdout around the call itself.
func newLocalTokenizerSilent(model string) (*tokenizer.LocalTokenizer, error) {
	// Redirect os.Stdout to /dev/null for the duration of this call so the
	// SDK's fmt.Println warning doesn't pollute the token-count output.
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		// If we can't open /dev/null just call through — worst case the warning
		// appears, which is better than silently failing.
		return tokenizer.NewLocalTokenizer(model)
	}
	orig := os.Stdout
	os.Stdout = devNull
	tok, tokErr := tokenizer.NewLocalTokenizer(model)
	os.Stdout = orig
	devNull.Close()
	return tok, tokErr
}

// geminiAlias maps a model name that genai/tokenizer doesn't recognize to a
// closely-related sibling that it does. The alias is chosen to share the same
// underlying vocab so token counts match what the real model would produce.
//
// Justification: google.golang.org/genai/tokenizer's tokenMap shows
// gemini-2.0-flash, gemini-2.5-{pro,flash,flash-lite}, and
// gemini-3-pro-preview all mapped to the "gemma3" sentencepiece vocab
// (gemma3_cleaned_262144_v2.spiece.model). The Gemini 3.1 Pro model card
// states "for architecture see Gemini 3 Pro", so the 3.x family is treated
// as a single tokenization family. We still keep the remote fallback for
// strict correctness when an API key is set.
//
// Returning "" means "no alias known — fall through to remote".
func geminiAlias(model string) string {
	switch {
	// Pro variants of the 3.x family (3, 3.1, future 3.x) — the canonical
	// gemma3 model in genai/tokenizer is gemini-3-pro-preview.
	case model == "gemini-3.1-pro-preview",
		model == "gemini-3-pro-preview",
		strings.HasPrefix(model, "gemini-3-pro"),
		strings.HasPrefix(model, "gemini-3.1-pro"):
		return "gemini-3-pro-preview"
	// Flash/Flash-Lite variants of the 3.x family — there's no flash-3 entry
	// in the SDK, so we route to gemini-2.5-flash which is also gemma3.
	case strings.HasPrefix(model, "gemini-3.1-flash"),
		strings.HasPrefix(model, "gemini-3-flash"),
		strings.HasPrefix(model, "gemini-3.1-flash-lite"):
		return "gemini-2.5-flash"
	}
	return ""
}

// googleProvider counts tokens for Google Gemini models using a layered strategy:
//  1. genai/tokenizer.NewLocalTokenizer + CountTokens (offline, no key).
//  2. Alias mapping to a known-equivalent gemma3 sibling for newer model
//     names the SDK doesn't yet list (gemini-3.1-*-preview, etc.).
//  3. Remote API fallback via genai.NewClient. Requires GEMINI_API_KEY or
//     GOOGLE_API_KEY. With (1)+(2) covering every model in the catalog as
//     of 2026-05, this path is rarely reached.
type googleProvider struct{}

func (p *googleProvider) Name() string { return "google" }

// tryLocal attempts a local count for the given model name and returns the
// result plus a "did it work" flag. It does NOT log or print on failure —
// callers fall through to the next layer.
func (p *googleProvider) tryLocal(model string, contents []*genai.Content) (int, bool) {
	tok, err := newLocalTokenizerSilent(model)
	if err != nil {
		return 0, false
	}
	result, err := tok.CountTokens(contents, nil)
	if err != nil {
		return 0, false
	}
	return int(result.TotalTokens), true
}

func (p *googleProvider) Count(ctx context.Context, model string, text string) (int, error) {
	contents := []*genai.Content{genai.NewContentFromText(text, genai.RoleUser)}

	// 1. Local tokenization with the model name as given.
	if n, ok := p.tryLocal(model, contents); ok {
		return n, nil
	}

	// 2. Alias retry — translate to a confirmed-equivalent gemma3 model.
	if alias := geminiAlias(model); alias != "" {
		if n, ok := p.tryLocal(alias, contents); ok {
			return n, nil
		}
	}

	// 3. Remote fallback: requires API key.
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return 0, fmt.Errorf("GEMINI_API_KEY (or GOOGLE_API_KEY) environment variable is required for Gemini remote tokenization")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return 0, fmt.Errorf("google: request failed: %w", err)
	}

	result, err := client.Models.CountTokens(ctx, model, contents, nil)
	if err != nil {
		return 0, fmt.Errorf("google: request failed: %w", err)
	}

	return int(result.TotalTokens), nil
}
