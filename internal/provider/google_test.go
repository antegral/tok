package provider

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestGoogleProviderName(t *testing.T) {
	p := &googleProvider{}
	if got := p.Name(); got != "google" {
		t.Errorf("Name() = %q, want %q", got, "google")
	}
}

// TestGoogleProviderRemoteFallback_NoKey verifies that when both GEMINI_API_KEY
// and GOOGLE_API_KEY are unset and the model name is not supported by the local
// tokenizer, Count returns the exact remote-fallback error message.
func TestGoogleProviderRemoteFallback_NoKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")

	p := &googleProvider{}
	_, err := p.Count(context.Background(), "definitely-not-a-real-model-xyz", "hello")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	want := "GEMINI_API_KEY (or GOOGLE_API_KEY) environment variable is required for Gemini remote tokenization"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
	}
}

// TestGoogleProviderIntegration is an optional integration test that calls the
// real Gemini API. It is skipped when neither GEMINI_API_KEY nor GOOGLE_API_KEY
// is set in the environment, or when -short is passed.
func TestGoogleProviderIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		t.Skip("neither GEMINI_API_KEY nor GOOGLE_API_KEY is set; skipping integration test")
	}

	p := &googleProvider{}
	count, err := p.Count(context.Background(), "gemini-1.5-flash", "Hello, world.")
	if err != nil {
		t.Fatalf("Count() error: %v", err)
	}
	if count <= 0 {
		t.Errorf("Count() = %d, want > 0", count)
	}
}
