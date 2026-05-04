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

// TestOpenAI_Count_UnknownModel_NoKey verifies that requesting a model unknown
// to tiktoken-go without an API key returns the exact error message expected by
// main.go (which prepends "error: " before printing).
func TestOpenAI_Count_UnknownModel_NoKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	p := &openaiProvider{}
	_, err := p.Count(context.Background(), "gpt-5.5", "hi")
	if err == nil {
		t.Fatal("expected error for unknown model without API key, got nil")
	}
	want := `OPENAI_API_KEY environment variable is required for openai model "gpt-5.5" (not in local tokenizer)`
	if err.Error() != want {
		t.Errorf("error message:\n  got:  %q\n  want: %q", err.Error(), want)
	}
}

// TestOpenAI_Count_UnknownModel_WithKey is an integration test that calls the
// real OpenAI API. It is skipped unless OPENAI_API_KEY is set in the environment.
func TestOpenAI_Count_UnknownModel_WithKey(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set; skipping integration test")
	}
	p := &openaiProvider{}
	got, err := p.Count(context.Background(), "gpt-5.5", "The quick brown fox jumps over the lazy dog.")
	if err != nil {
		t.Fatalf("Count(gpt-5.5, fox) error: %v", err)
	}
	if got <= 0 {
		t.Errorf("Count(gpt-5.5, fox) = %d, want > 0", got)
	}
	t.Logf("gpt-5.5 token count for fox text: %d", got)
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
