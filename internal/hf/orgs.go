// Package hf provides AUTOCOMPLETE-ONLY helpers that query the public
// HuggingFace Hub API to enumerate models for a given org. It is NOT used
// for tokenization (the daulet/tokenizers library handles that path) and
// must therefore stay best-effort: any error returns silently to the
// caller so the shell completion experience is never blocked by network
// or auth failures.
package hf

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// hfAPIURL is the JSON model-listing endpoint. We sort by downloads so the
// most useful candidates appear first in shell completion menus.
const hfAPIURL = "https://huggingface.co/api/models?author=%s&limit=50&sort=downloads&direction=-1"

// cacheTTL bounds how stale a per-org model list may be before we refresh
// it from the API. 24h matches the plan and is plenty for autocomplete.
const cacheTTL = 24 * time.Hour

// fetchTimeout caps how long a single HF API call may block shell
// completion. Anything slower than this is treated as "no candidates" so
// the user's Tab key never feels frozen.
const fetchTimeout = 2 * time.Second

// orgCache is the on-disk shape of the per-org model list cache.
// Models are stored in their canonical "<org>/<repo>" form so the caller
// can use them directly as completion candidates.
type orgCache struct {
	Fetched time.Time `json:"fetched"`
	Models  []string  `json:"models"`
}

// FetchOrgModels returns "<org>/<repo>" identifiers for the given HF org.
// On any error it returns (nil, err) — callers should treat errors as
// "no candidates" rather than propagating them to the user.
//
// Caching: results are persisted to ~/.cache/tok/hf-orgs/<org>.json with
// a 24h TTL. Cache hits never touch the network.
func FetchOrgModels(org string) ([]string, error) {
	cachePath := cacheFilePath(org)

	// Cache hit + still fresh → return immediately, no network call.
	if data, err := os.ReadFile(cachePath); err == nil {
		var c orgCache
		if json.Unmarshal(data, &c) == nil && time.Since(c.Fetched) < cacheTTL {
			return c.Models, nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	apiURL := fmt.Sprintf(hfAPIURL, url.QueryEscape(org))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	if tok := os.Getenv("HF_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hf api: status code %d", resp.StatusCode)
	}

	var raw []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(raw))
	for _, r := range raw {
		if r.ID != "" {
			ids = append(ids, r.ID)
		}
	}

	// Persist cache. Failures to write are non-fatal — the user gets the
	// fresh result this turn and we'll just refetch next time.
	if err := writeCache(cachePath, ids); err != nil {
		// Intentionally swallowed: the call succeeded, the cache write
		// did not. The next completion will retry. We do NOT log to
		// stderr because that would corrupt non-completion CLI output.
		_ = err
	}

	return ids, nil
}

// cacheFilePath returns the absolute path of the per-org cache file.
// It does NOT create the directory — writeCache handles that on demand.
func cacheFilePath(org string) string {
	return filepath.Join(cacheDir(), org+".json")
}

// cacheDir returns the directory holding all per-org cache files.
// Layout: <UserCacheDir>/tok/hf-orgs/. Falls back to TempDir if the user
// cache dir is unavailable.
func cacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "tok", "hf-orgs")
}

// writeCache persists the given model list to cachePath atomically (write
// to a temp file in the same directory then rename) so a partial write
// never poisons future cache reads.
func writeCache(cachePath string, ids []string) error {
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(orgCache{Fetched: time.Now(), Models: ids})
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(cachePath), "orgcache-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, cachePath)
}
