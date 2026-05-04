package provider

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestHF_Name(t *testing.T) {
	p := &hfProvider{}
	if got := p.Name(); got != "hf" {
		t.Errorf("Name() = %q, want %q", got, "hf")
	}
}

func TestHF_isAuthError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("boom"), false},
		{errors.New("download failed: status code 401"), true},
		{errors.New("download failed: status code 403"), true},
		{errors.New("download failed: status code 404"), false},
		{errors.New("status code 500"), false},
	}
	for _, tc := range cases {
		var label string
		if tc.err != nil {
			label = tc.err.Error()
		} else {
			label = "<nil>"
		}
		t.Run(label, func(t *testing.T) {
			if got := isAuthError(tc.err); got != tc.want {
				t.Errorf("isAuthError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestHF_isNotFoundError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("status code 404"), true},
		{errors.New("status code 401"), false},
		{errors.New("not the right shape"), false},
	}
	for _, tc := range cases {
		var label string
		if tc.err != nil {
			label = tc.err.Error()
		} else {
			label = "<nil>"
		}
		t.Run(label, func(t *testing.T) {
			if got := isNotFoundError(tc.err); got != tc.want {
				t.Errorf("isNotFoundError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// TestHF_Count_Integration exercises the real HuggingFace download +
// tokenize path against a small, well-known public model. It is gated by
// TOK_HF_INTEGRATION=1 to keep the default `go test ./...` run hermetic
// (no network, no ~hundreds-of-MB download). The model used —
// google-bert/bert-base-uncased — is the same one daulet/tokenizers'
// own README uses, and its tokenizer.json is small (~700KB).
func TestHF_Count_Integration(t *testing.T) {
	if os.Getenv("TOK_HF_INTEGRATION") != "1" {
		t.Skip("TOK_HF_INTEGRATION not set — skipping HF integration test")
	}
	p := &hfProvider{}
	got, err := p.Count(context.Background(), "google-bert/bert-base-uncased", "Hello, world.")
	if err != nil {
		t.Fatalf("Count() error: %v", err)
	}
	if got <= 0 {
		t.Fatalf("Count() = %d, want > 0", got)
	}
	t.Logf("bert-base-uncased token count for 'Hello, world.' = %d", got)
}

// TestHF_Count_NotFound checks that a clearly non-existent repo produces
// an error that mentions the model name (the user needs to see WHICH
// model failed). Network is required for this test, so it is also gated.
func TestHF_Count_NotFound(t *testing.T) {
	if os.Getenv("TOK_HF_INTEGRATION") != "1" {
		t.Skip("TOK_HF_INTEGRATION not set — skipping HF integration test")
	}
	p := &hfProvider{}
	_, err := p.Count(context.Background(), "definitely-fake-org-xyz/definitely-fake-repo", "hi")
	if err == nil {
		t.Fatal("expected error for fake model, got nil")
	}
	if !strings.Contains(err.Error(), "definitely-fake-org-xyz/definitely-fake-repo") {
		t.Errorf("error should mention the model name, got: %v", err)
	}
}
