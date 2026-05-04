package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daulet/tokenizers"
)

// hfProvider counts tokens via daulet/tokenizers, which downloads a model's
// tokenizer.json from the HuggingFace Hub on first use and tokenizes locally
// using a Rust-based tokenizer (linked statically as libtokenizers.a).
//
// The model parameter passed to Count is the FULL "<org>/<repo>" spec
// because tokenizers.FromPretrained expects that exact format and registry
// routing intentionally hands HF the whole spec (see registry.Resolve).
type hfProvider struct{}

func (p *hfProvider) Name() string { return "hf" }

// hfCacheDir returns a stable on-disk cache directory for downloaded
// tokenizer artifacts. We prefer the user cache dir (~/.cache/tok/hf on
// Linux, ~/Library/Caches/tok/hf on macOS) and fall back to TempDir if
// the user cache dir is unavailable for any reason.
func hfCacheDir() string {
	if d, err := os.UserCacheDir(); err == nil {
		return filepath.Join(d, "tok", "hf")
	}
	return filepath.Join(os.TempDir(), "tok", "hf")
}

// isAuthError returns true if err appears to be an HF Hub 401/403 — the
// upstream library wraps the HTTP failure as a string ("status code 401"),
// so substring matching is the only available signal.
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "status code 401") || strings.Contains(msg, "status code 403")
}

// isNotFoundError returns true if err looks like a 404 from the Hub.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "status code 404")
}

func (p *hfProvider) Count(ctx context.Context, model string, text string) (int, error) {
	// Build options: cache dir is mandatory (otherwise the library uses a
	// throwaway tempdir and re-downloads on every invocation), auth token
	// is added only when the user has set HF_TOKEN.
	opts := []tokenizers.TokenizerConfigOption{
		tokenizers.WithCacheDir(hfCacheDir()),
	}
	hfToken := os.Getenv("HF_TOKEN")
	if hfToken != "" {
		opts = append(opts, tokenizers.WithAuthToken(hfToken))
	}

	tk, err := tokenizers.FromPretrained(model, opts...)
	if err != nil {
		// Translate the most user-actionable upstream errors to the exact
		// messages mandated by implementation-plan.md §10.
		switch {
		case isAuthError(err) && hfToken == "":
			return 0, fmt.Errorf("HF_TOKEN environment variable is required for private model %s", model)
		case isAuthError(err):
			return 0, fmt.Errorf("hf: authentication failed for model %q (check HF_TOKEN): %w", model, err)
		case isNotFoundError(err):
			return 0, fmt.Errorf("hf: model %q not found", model)
		default:
			return 0, fmt.Errorf("hf: failed to load tokenizer for %q: %w", model, err)
		}
	}
	if tk == nil {
		return 0, errors.New("hf: tokenizer is nil")
	}
	defer tk.Close()

	// addSpecialTokens=false: we count the raw user text only, no [CLS]/[SEP]
	// or equivalent decoration that would inflate counts versus what an LLM
	// would actually consume for the same input.
	ids, _ := tk.Encode(text, false)
	return len(ids), nil
}
