package provider

import (
	"fmt"
	"strings"
)

// Resolve maps a model spec string to a concrete Provider and the model name
// that should be passed to that Provider.
//
// Routing rules (per implementation-plan.md §0, §4):
//   - "openai/<model>"    → openaiProvider, model = "<model>"
//   - "anthropic/<model>" → anthropicProvider, model = "<model>"
//   - "google/<model>"    → googleProvider, model = "<model>"
//   - everything else     → hfProvider, model = full spec ("<org>/<repo>")
//
// The default branch covers all non-known prefixes so unknown providers fall
// through to HuggingFace. There is intentionally no "unknown provider" error.
//
// Returns an error if the spec does not contain "<x>/<y>" with a non-empty y.
func Resolve(spec string) (Provider, string, error) {
	parts := strings.SplitN(spec, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, "", fmt.Errorf("invalid model spec %q (expected <provider>/<model> or <hf-org>/<repo>)", spec)
	}
	switch parts[0] {
	case "openai":
		return &openaiProvider{}, parts[1], nil
	case "anthropic":
		return &anthropicProvider{}, parts[1], nil
	case "google":
		return &googleProvider{}, parts[1], nil
	default:
		// All non-known prefixes route to HuggingFace; the model name is the
		// full spec because tokenizers.FromPretrained expects "<org>/<repo>".
		return &hfProvider{}, spec, nil
	}
}
