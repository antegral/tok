package provider

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestAnthropic_Name(t *testing.T) {
	p := &anthropicProvider{}
	if p.Name() != "anthropic" {
		t.Errorf("Name() = %q, want %q", p.Name(), "anthropic")
	}
}

func TestAnthropic_Count_MissingKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")

	p := &anthropicProvider{}
	_, err := p.Count(context.Background(), "claude-sonnet-4-5", "Hello, world.")
	if err == nil {
		t.Fatal("expected error when ANTHROPIC_API_KEY is unset, got nil")
	}
	want := "ANTHROPIC_API_KEY environment variable is required for Claude models"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error message: got %q, want substring %q", err.Error(), want)
	}
}

// TestAnthropic_Count_Integration runs a real token-count API call.
// It is skipped when ANTHROPIC_API_KEY is not set in the environment.
func TestAnthropic_Count_Integration(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set — skipping integration test")
	}

	p := &anthropicProvider{}
	got, err := p.Count(context.Background(), "claude-haiku-4-5", "Hello, world.")
	if err != nil {
		t.Fatalf("Count() error: %v", err)
	}
	if got <= 0 {
		t.Errorf("Count() = %d, want > 0", got)
	}
	t.Logf("claude-haiku-4-5 token count for 'Hello, world.' = %d", got)
}
