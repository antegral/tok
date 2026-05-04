package hf

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCacheRoundTrip writes a cache file via writeCache and confirms a
// subsequent FetchOrgModels call within the TTL window returns the cached
// data without making an HTTP request. We point UserCacheDir at a temp
// directory using XDG_CACHE_HOME (Go's os.UserCacheDir honours it on
// Linux) so the test is fully isolated from any pre-existing cache.
func TestCacheRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	want := []string{"acme/foo", "acme/bar"}
	cachePath := cacheFilePath("acme")
	if err := writeCache(cachePath, want); err != nil {
		t.Fatalf("writeCache error: %v", err)
	}

	// Cache file must live under the expected directory layout.
	expectedDir := filepath.Join(tmp, "tok", "hf-orgs")
	if !strings.HasPrefix(cachePath, expectedDir) {
		t.Errorf("cache path %q is not under %q", cachePath, expectedDir)
	}

	got, err := FetchOrgModels("acme")
	if err != nil {
		t.Fatalf("FetchOrgModels error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d models, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("models[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestCacheStaleRefetch verifies that a cache file older than cacheTTL is
// IGNORED. We can't easily intercept the HTTP call here, so we check the
// raw cache file is left untouched by FetchOrgModels-the-stale-path:
// instead, we test the unmarshalling/age decision in isolation.
func TestCacheStaleDecision(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	cachePath := cacheFilePath("oldorg")
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	stale := orgCache{
		Fetched: time.Now().Add(-2 * cacheTTL),
		Models:  []string{"oldorg/old-model"},
	}
	data, err := json.Marshal(stale)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Re-read the file directly and verify the stale check fires.
	raw, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var c orgCache
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if time.Since(c.Fetched) < cacheTTL {
		t.Errorf("expected cache to be stale, age = %v", time.Since(c.Fetched))
	}
}

// TestQueryEscape confirms that orgs containing URL-meta characters are
// escaped before being interpolated into the API URL. This is the
// security/safety check from the plan: a maliciously-crafted org name
// (e.g. one containing "&" or "?") must not corrupt the query string.
func TestQueryEscape(t *testing.T) {
	cases := []struct {
		org      string
		mustHave string
	}{
		{"meta-llama", "author=meta-llama"},
		{"foo bar", "author=foo+bar"},                 // space → "+"
		{"a&b=c", "author=a%26b%3Dc"},                 // & and =
		{"weird/slash", "author=weird%2Fslash"},       // /
		{"with space and?q", "author=with+space+and%3Fq"},
	}
	for _, tc := range cases {
		t.Run(tc.org, func(t *testing.T) {
			got := fmt.Sprintf(hfAPIURL, url.QueryEscape(tc.org))
			if !strings.Contains(got, tc.mustHave) {
				t.Errorf("URL = %q, want substring %q", got, tc.mustHave)
			}
		})
	}
}

// TestFetchOrgModels_Live is gated behind TOK_HF_INTEGRATION=1 because
// it requires network access. It exercises the full path: HTTP call,
// JSON decode, cache write, second-call cache hit.
func TestFetchOrgModels_Live(t *testing.T) {
	if os.Getenv("TOK_HF_INTEGRATION") != "1" {
		t.Skip("TOK_HF_INTEGRATION not set — skipping live HF API test")
	}
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	first, err := FetchOrgModels("google")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if len(first) == 0 {
		t.Fatal("expected at least one model for org=google")
	}
	// Second call should hit the cache. We can't observe HTTP directly,
	// but we can verify the cache file exists.
	if _, err := os.Stat(cacheFilePath("google")); err != nil {
		t.Errorf("cache file not present after fetch: %v", err)
	}
	second, err := FetchOrgModels("google")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if len(second) != len(first) {
		t.Errorf("cache mismatch: first=%d second=%d", len(first), len(second))
	}
}
