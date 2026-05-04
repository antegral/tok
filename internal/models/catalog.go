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
//
// Source: each provider's /v1/models endpoint as of 2026-05. See
// .omc/plans/implementation-plan.md §1.1/§1.3 for fallback behavior — every
// model below tokenizes locally either directly (tiktoken-go for OpenAI,
// genai/tokenizer for Gemini) or via prefix/alias fallback. Anthropic still
// requires the remote count_tokens API because no public Claude tokenizer
// exists.
//
// Variants intentionally dropped from the live API output (per user
// preference): -chat-latest aliases and -codex-* models on OpenAI side.
var BuiltinModels = map[string][]string{
	"openai": {
		"openai/gpt-5.5",
		"openai/gpt-5.5-pro",
		"openai/gpt-5.4",
		"openai/gpt-5.4-mini",
		"openai/gpt-5.4-nano",
		"openai/gpt-5.4-pro",
		"openai/gpt-5.2",
		"openai/gpt-5.2-pro",
		"openai/gpt-5.1",
		"openai/gpt-5",
		"openai/gpt-5-mini",
		"openai/gpt-5-nano",
		"openai/gpt-5-pro",
		"openai/gpt-4.1",
		"openai/gpt-4.1-mini",
		"openai/gpt-4.1-nano",
		"openai/gpt-4o",
		"openai/gpt-4o-mini",
		"openai/o3",
		"openai/o3-mini",
		"openai/o4-mini",
		"openai/o1",
		"openai/o1-pro",
	},
	"anthropic": {
		"anthropic/claude-opus-4-7",
		"anthropic/claude-opus-4-6",
		"anthropic/claude-opus-4-5",
		"anthropic/claude-sonnet-4-6",
		"anthropic/claude-sonnet-4-5",
		"anthropic/claude-haiku-4-5",
		"anthropic/claude-opus-4-1",
	},
	"google": {
		"google/gemini-3.1-pro-preview",
		"google/gemini-3.1-flash-lite-preview",
		"google/gemini-3-pro-preview",
		"google/gemini-3-flash-preview",
		"google/gemini-2.5-pro",
		"google/gemini-2.5-flash",
		"google/gemini-2.5-flash-lite",
		"google/gemini-2.0-flash",
		"google/gemini-2.0-flash-lite",
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
