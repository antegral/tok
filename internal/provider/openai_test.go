package provider

import (
	"context"
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

func TestOpenAI_Count_UnknownModel(t *testing.T) {
	p := &openaiProvider{}
	_, err := p.Count(context.Background(), "definitely-not-a-real-model-xyz", "hi")
	if err == nil {
		t.Fatal("expected error for unknown model, got nil")
	}
	want := `openai: unknown model "definitely-not-a-real-model-xyz"`
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error message: got %q, want substring %q", err.Error(), want)
	}
}

func TestOpenAI_Name(t *testing.T) {
	p := &openaiProvider{}
	if p.Name() != "openai" {
		t.Errorf("Name() = %q, want %q", p.Name(), "openai")
	}
}
