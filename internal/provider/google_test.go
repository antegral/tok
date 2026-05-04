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

// TestGoogleProvider_LocalAlias_NoKey verifies that 2026-05 catalog models
// the SDK does NOT directly know (gemini-3.1-pro-preview, gemini-3-flash-preview,
// gemini-3.1-flash-lite-preview) still tokenize locally via the alias mapping
// in google.go. No API key required.
func TestGoogleProvider_LocalAlias_NoKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping vocab-download test in short mode")
	}
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")

	p := &googleProvider{}
	const text = "Hello, world."

	models := []string{
		"gemini-3.1-pro-preview",
		"gemini-3-pro-preview",
		"gemini-3-flash-preview",
		"gemini-3.1-flash-lite-preview",
	}
	for _, m := range models {
		t.Run(m, func(t *testing.T) {
			got, err := p.Count(context.Background(), m, text)
			if err != nil {
				t.Fatalf("Count(%q) error: %v (expected local alias fallback)", m, err)
			}
			if got <= 0 {
				t.Errorf("Count(%q) = %d, want > 0", m, got)
			}
		})
	}
}

// TestGoogleProvider_LocalDirect_NoKey covers a model that the SDK accepts
// directly without aliasing. Should still tokenize offline.
func TestGoogleProvider_LocalDirect_NoKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping vocab-download test in short mode")
	}
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")

	p := &googleProvider{}
	got, err := p.Count(context.Background(), "gemini-1.5-pro", "Hello, world.")
	if err != nil {
		t.Fatalf("Count(gemini-1.5-pro) error: %v", err)
	}
	if got <= 0 {
		t.Errorf("Count(gemini-1.5-pro) = %d, want > 0", got)
	}
}

// TestGoogleProviderRemoteFallback_NoKey verifies that when the model is
// unknown to BOTH the local tokenizer and the alias map, and no API key is
// available, Count returns the exact remote-fallback error message.
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
