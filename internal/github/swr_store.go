package github

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/cli/go-gh/v2/pkg/config"
	"github.com/gertzgal/gh-prs/internal/filter"
	"github.com/gertzgal/gh-prs/internal/model"
)

const swrVersion = 2

// swrStore handles disk I/O for the stale-while-revalidate cache.
type swrStore struct {
	baseDir string
}

func newSWRStore(baseDir string) *swrStore {
	return &swrStore{baseDir: baseDir}
}

// path returns the cache file path for a given account + repo + query filter set.
func (s *swrStore) path(accountID, owner, name, filterKey string) string {
	safe := func(s string) string { return strings.ReplaceAll(s, "/", "_") }
	return filepath.Join(
		s.baseDir,
		"swr",
		safe(accountID),
		fmt.Sprintf("%s_%s_%s.json", safe(owner), safe(name), filterKey),
	)
}

// read loads an entry from disk. Returns (nil, nil) if the file does not exist
// or the version is stale.
func (s *swrStore) read(accountID, owner, name, filterKey string) (*swrEntry, error) {
	p := s.path(accountID, owner, name, filterKey)
	raw, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entry swrEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return nil, nil // corrupt file, treat as miss
	}
	if entry.Version != swrVersion {
		return nil, nil // old format, treat as miss
	}
	if entry.Data == nil {
		return nil, nil
	}
	return &entry, nil
}

// write persists an entry to disk.
func (s *swrStore) write(accountID, owner, name, filterKey string, repo *model.Repo) error {
	p := s.path(accountID, owner, name, filterKey)
	if err := os.MkdirAll(filepath.Dir(p), 0750); err != nil {
		return err
	}
	entry := swrEntry{
		Version:   swrVersion,
		WrittenAt: time.Now(),
		Data:      repo,
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(p, raw, 0640)
}

func cacheFilterKey(filters filter.Set) string {
	frags := append([]string(nil), filters.QueryFragments()...)
	if len(frags) == 0 {
		return "all"
	}
	sort.Strings(frags)
	sum := sha256.Sum256([]byte(strings.Join(frags, "\x00")))
	return fmt.Sprintf("%x", sum)[:16]
}

// accountID returns a stable identifier for the current GitHub account.
// It prefers the viewer login from gh config; falls back to a token hash.
func accountID() string {
	cfg, err := config.Read(nil)
	if err == nil {
		login, err := cfg.Get([]string{"hosts", "github.com", "user"})
		if err == nil && login != "" {
			return login
		}
	}
	token, _ := auth.TokenForHost("github.com")
	if token != "" {
		h := sha256.Sum256([]byte(token))
		return fmt.Sprintf("%x", h)[:16]
	}
	return "unknown"
}

// swrEntry is the on-disk envelope for cached repo data.
type swrEntry struct {
	Version   int         `json:"v"`
	WrittenAt time.Time   `json:"t"`
	Data      *model.Repo `json:"data"`
}
