// Package models holds the hard-coded catalog of known-provider models used
// for tab completion, plus the CompleteSpec function that drives Cobra's
// ValidArgsFunction.
package models

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/antegral/tok/internal/hf"
)

// BuiltinModels is the hard-coded catalog of known-provider models exposed
// to shell completion. HuggingFace models are intentionally absent — they
// are looked up dynamically by org via the HF API in a later phase.
var BuiltinModels = map[string][]string{
	"openai": {
		"openai/gpt-5.5", "openai/gpt-5.5-pro",
		"openai/gpt-5", "openai/gpt-5-mini", "openai/gpt-5-nano",
		"openai/gpt-4.1", "openai/gpt-4o", "openai/gpt-4o-mini",
		"openai/o3", "openai/o4-mini",
	},
	"anthropic": {
		"anthropic/claude-opus-4-7",
		"anthropic/claude-opus-4-6",
		"anthropic/claude-sonnet-4-6",
		"anthropic/claude-haiku-4-5",
	},
	"google": {
		"google/gemini-3.1-pro",
		"google/gemini-3.1-flash-lite",
		"google/gemini-3-flash",
		"google/gemini-2.5-pro", "google/gemini-2.5-flash",
	},
}

// knownProviderPrefixes is the ordered list of provider prefixes shown when
// the user has not yet typed a slash. Order is the display order in shell.
var knownProviderPrefixes = []string{"openai/", "google/", "anthropic/"}

// CompleteSpec implements the 3-stage completion logic from plan §6.3:
//
//  1. No slash in toComplete → suggest known provider prefixes
//     (e.g. "openai/", "google/", "anthropic/") with NoSpace so the next
//     Tab triggers model completion.
//  2. Slash present and prefix matches a known provider → suggest models
//     from BuiltinModels filtered by toComplete prefix.
//  3. Slash present and prefix is unknown (HuggingFace org case) →
//     query the HF Hub for models in that org via hf.FetchOrgModels.
//     Errors are swallowed silently (best-effort autocomplete).
func CompleteSpec(toComplete string) ([]string, cobra.ShellCompDirective) {
	if !strings.Contains(toComplete, "/") {
		var out []string
		for _, p := range knownProviderPrefixes {
			if strings.HasPrefix(p, toComplete) {
				out = append(out, p)
			}
		}
		return out, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	}

	parts := strings.SplitN(toComplete, "/", 2)
	switch parts[0] {
	case "openai", "anthropic", "google":
		return prefixMatch(BuiltinModels[parts[0]], toComplete), cobra.ShellCompDirectiveNoFileComp
	default:
		// HF org dynamic lookup. Errors → empty candidates (silent),
		// per plan §6.6: a flaky network must never break completion.
		hfModels, err := hf.FetchOrgModels(parts[0])
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return prefixMatch(hfModels, toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}

// prefixMatch returns entries of items that have toComplete as a prefix.
// A nil/empty result is fine — Cobra handles it gracefully.
func prefixMatch(items []string, toComplete string) []string {
	var out []string
	for _, it := range items {
		if strings.HasPrefix(it, toComplete) {
			out = append(out, it)
		}
	}
	return out
}
