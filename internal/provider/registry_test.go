package provider

import (
	"strings"
	"testing"
)

func TestResolve_KnownProviders(t *testing.T) {
	cases := []struct {
		spec       string
		wantName   string
		wantModel  string
	}{
		{"openai/gpt-4o", "openai", "gpt-4o"},
		{"openai/gpt-3.5-turbo", "openai", "gpt-3.5-turbo"},
		{"anthropic/claude-sonnet-4-5", "anthropic", "claude-sonnet-4-5"},
		{"google/gemini-1.5-pro", "google", "gemini-1.5-pro"},
	}
	for _, tc := range cases {
		t.Run(tc.spec, func(t *testing.T) {
			p, m, err := Resolve(tc.spec)
			if err != nil {
				t.Fatalf("Resolve(%q) returned error: %v", tc.spec, err)
			}
			if p.Name() != tc.wantName {
				t.Errorf("provider name: got %q, want %q", p.Name(), tc.wantName)
			}
			if m != tc.wantModel {
				t.Errorf("model: got %q, want %q", m, tc.wantModel)
			}
		})
	}
}

func TestResolve_HFFallthrough(t *testing.T) {
	// Any prefix that isn't openai/anthropic/google routes to HF and the
	// model passed downstream is the FULL spec ("<org>/<repo>").
	cases := []struct {
		spec      string
		wantModel string
	}{
		{"meta-llama/Llama-3.1-8B-Instruct", "meta-llama/Llama-3.1-8B-Instruct"},
		{"foo/bar", "foo/bar"},
		{"BAAI/bge-small-en", "BAAI/bge-small-en"},
		// Misspelled "known" provider also falls through to HF.
		{"openai-foo/bar", "openai-foo/bar"},
	}
	for _, tc := range cases {
		t.Run(tc.spec, func(t *testing.T) {
			p, m, err := Resolve(tc.spec)
			if err != nil {
				t.Fatalf("Resolve(%q) returned error: %v", tc.spec, err)
			}
			if p.Name() != "hf" {
				t.Errorf("provider name: got %q, want %q", p.Name(), "hf")
			}
			if m != tc.wantModel {
				t.Errorf("model: got %q, want %q", m, tc.wantModel)
			}
		})
	}
}

func TestResolve_InvalidSpecs(t *testing.T) {
	cases := []string{
		"",        // empty
		"foobar",  // no slash
		"openai/", // empty model
		"/gpt-4o", // empty provider
		"/",       // empty both
	}
	for _, spec := range cases {
		t.Run(spec, func(t *testing.T) {
			p, m, err := Resolve(spec)
			if err == nil {
				t.Fatalf("Resolve(%q): expected error, got provider=%v model=%q", spec, p, m)
			}
			if !strings.Contains(err.Error(), "invalid model spec") {
				t.Errorf("error message should mention 'invalid model spec', got: %v", err)
			}
		})
	}
}
