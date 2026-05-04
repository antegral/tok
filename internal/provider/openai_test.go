package provider

import (
	"context"
	"os"
	"strings"
	"testing"
)

// Token counts here were obtained from tiktoken-go v0.7.0 directly and are
// also consistent with OpenAI's reference tokenizer for the cl100k/o200k
// encodings used by these models.
func TestOpenAI_Count_KnownStrings(t *testing.T) {
	p := &openaiProvider{}
	cases := []struct {
		model string
		text  string
		want  int
	}{
		{"gpt-4o", "Hello, world.", 4},
		{"gpt-4o", "The quick brown fox jumps over the lazy dog.", 10},
		{"gpt-4o", "", 0},
		{"gpt-4o", "abc", 1},
		{"gpt-3.5-turbo", "Hello, world.", 4},
		// Versioned variant resolves via tiktoken-go's internal prefix table.
		{"gpt-4o-2024-08-06", "Hello, world.", 4},
	}
	for _, tc := range cases {
		t.Run(tc.model+"::"+tc.text, func(t *testing.T) {
			got, err := p.Count(context.Background(), tc.model, tc.text)
			if err != nil {
				t.Fatalf("Count(%q, %q) error: %v", tc.model, tc.text, err)
			}
			if got != tc.want {
				t.Errorf("Count(%q, %q) = %d, want %d", tc.model, tc.text, got, tc.want)
			}
		})
	}
}

// TestOpenAI_Count_PrefixFallback_NoKey verifies that 2026-05 models which
// tiktoken-go does NOT directly know (gpt-5.5, gpt-5.5-pro, gpt-5.4,
// gpt-5.4-pro) still tokenize locally via openai.go's prefix-matching
// fallback to o200k_base. No OPENAI_API_KEY required.
func TestOpenAI_Count_PrefixFallback_NoKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	p := &openaiProvider{}

	const text = "The quick brown fox jumps over the lazy dog."
	// Reference count for the o200k_base encoding (same as gpt-4o for the
	// same input, established in TestOpenAI_Count_KnownStrings above).
	const want = 10

	models := []string{
		"gpt-5.5",
		"gpt-5.5-pro",
		"gpt-5.4",
		"gpt-5.4-pro",
		"gpt-5.4-mini",
		"gpt-5.4-nano",
		"gpt-5.2",
		"gpt-5.2-pro",
		"gpt-5.1",
		"gpt-5-pro",
	}
	for _, m := range models {
		t.Run(m, func(t *testing.T) {
			got, err := p.Count(context.Background(), m, text)
			if err != nil {
				t.Fatalf("Count(%q) error: %v (expected local prefix fallback)", m, err)
			}
			if got <= 0 {
				t.Errorf("Count(%q) = %d, want > 0", m, got)
			}
			if got != want {
				t.Errorf("Count(%q) = %d, want %d (o200k_base reference for fox text)", m, got, want)
			}
		})
	}
}

// TestOpenAI_Count_Cl100kPrefixFallback_NoKey checks the cl100k_base side of
// the prefix matcher — gpt-4-32k is no longer in tiktoken-go's hard-coded
// model list but matches the gpt-4 prefix and should tokenize locally.
func TestOpenAI_Count_Cl100kPrefixFallback_NoKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	p := &openaiProvider{}
	got, err := p.Count(context.Background(), "gpt-4-32k", "Hello, world.")
	if err != nil {
		t.Fatalf("Count(gpt-4-32k) error: %v", err)
	}
	if got <= 0 {
		t.Errorf("Count(gpt-4-32k) = %d, want > 0", got)
	}
}

// TestOpenAI_Count_NoMatch_NoKey verifies that requesting a model that
// matches NEITHER tiktoken-go nor any prefix rule, without an API key,
// returns the exact error message expected by main.go (which prepends
// "error: " before printing).
func TestOpenAI_Count_NoMatch_NoKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	p := &openaiProvider{}
	const model = "definitely-not-a-real-model-xyz"
	_, err := p.Count(context.Background(), model, "hi")
	if err == nil {
		t.Fatal("expected error for unknown model without API key, got nil")
	}
	want := `OPENAI_API_KEY environment variable is required for openai model "` + model + `" (not in local tokenizer)`
	if err.Error() != want {
		t.Errorf("error message:\n  got:  %q\n  want: %q", err.Error(), want)
	}
}

// TestOpenAI_Count_RemoteFallback_WithKey is an integration test that calls
// the real OpenAI API. Skipped unless OPENAI_API_KEY is set. Uses a model
// name that matches no prefix so the remote path is actually exercised.
func TestOpenAI_Count_RemoteFallback_WithKey(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set; skipping integration test")
	}
	p := &openaiProvider{}
	// "babbage-002" exists in the OpenAI API but is not in tiktoken-go's
	// modern model list nor any of our prefix rules.
	got, err := p.Count(context.Background(), "babbage-002", "The quick brown fox jumps over the lazy dog.")
	if err != nil {
		t.Fatalf("Count(babbage-002) error: %v", err)
	}
	if got <= 0 {
		t.Errorf("Count(babbage-002) = %d, want > 0", got)
	}
	t.Logf("babbage-002 token count for fox text: %d", got)
}

func TestOpenAI_Name(t *testing.T) {
	p := &openaiProvider{}
	if p.Name() != "openai" {
		t.Errorf("Name() = %q, want %q", p.Name(), "openai")
	}
}

// Ensure the old unknown-model error string is gone (it's now superseded by
// the remote-fallback path).
func TestOpenAI_OldErrorGone(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	p := &openaiProvider{}
	_, err := p.Count(context.Background(), "definitely-not-a-real-model-xyz", "hi")
	if err == nil {
		t.Fatal("expected error for unknown model, got nil")
	}
	// Old error format must not appear.
	if strings.Contains(err.Error(), "unknown openai model") {
		t.Errorf("old error format still present: %q", err.Error())
	}
	// New format must appear.
	if !strings.Contains(err.Error(), "OPENAI_API_KEY environment variable is required") {
		t.Errorf("new error format missing: %q", err.Error())
	}
}
