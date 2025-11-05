package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gcli2api-go/internal/config"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

var (
	errGitUnsupported = &ErrNotSupported{Operation: "git_backend_operation"}
)

// GitOptions captures configuration required to operate a Git-backed storage.
type GitOptions struct {
	Path        string
	RemoteURL   string
	Branch      string
	Username    string
	Password    string
	AuthorName  string
	AuthorEmail string
}

// NewGitBackendFromConfig constructs a Git backend from configuration.
func NewGitBackendFromConfig(cfg *config.Config) *GitBackend {
	opts := GitOptions{
		Path:        expandPath(cfg.StorageBaseDir),
		RemoteURL:   strings.TrimSpace(cfg.GitRemoteURL),
		Branch:      strings.TrimSpace(cfg.GitBranch),
		Username:    strings.TrimSpace(cfg.GitUsername),
		Password:    strings.TrimSpace(cfg.GitPassword),
		AuthorName:  strings.TrimSpace(cfg.GitAuthorName),
		AuthorEmail: strings.TrimSpace(cfg.GitAuthorEmail),
	}
	if opts.Branch == "" {
		opts.Branch = "main"
	}
	if opts.Path == "" {
		opts.Path = filepath.Join(defaultBaseDir(), "git")
	}
	return NewGitBackend(opts)
}

// NewGitBackend creates a new Git-backed storage backend.
func NewGitBackend(opts GitOptions) *GitBackend {
	return &GitBackend{
		options: opts,
	}
}

// GitBackend implements storage.Backend using a Git repository as persistence layer.
type GitBackend struct {
	mu       sync.Mutex
	repo     *git.Repository
	worktree *git.Worktree
	options  GitOptions
}

const (
	gitCredentialDir = "credentials"
	gitConfigDir     = "config"
)

// Initialize prepares the git repository (clone or init).
func (g *GitBackend) Initialize(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := os.MkdirAll(g.options.Path, 0o755); err != nil {
		return fmt.Errorf("git backend: create base dir: %w", err)
	}

	var (
		repo *git.Repository
		err  error
	)

	if g.isExistingRepo() {
		repo, err = git.PlainOpen(g.options.Path)
		if err != nil {
			return fmt.Errorf("git backend: open existing repo: %w", err)
		}
	} else if g.options.RemoteURL != "" {
		repo, err = git.PlainClone(g.options.Path, false, &git.CloneOptions{
			URL:           g.options.RemoteURL,
			ReferenceName: plumbing.NewBranchReferenceName(g.options.Branch),
			SingleBranch:  true,
			Depth:         1,
			Auth:          g.auth(),
		})
		if err != nil {
			return fmt.Errorf("git backend: clone remote repo: %w", err)
		}
	} else {
		repo, err = git.PlainInit(g.options.Path, false)
		if err != nil {
			return fmt.Errorf("git backend: init repo: %w", err)
		}
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("git backend: worktree: %w", err)
	}

	if err := worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(g.options.Branch),
		Create: true,
	}); err != nil && !errors.Is(err, plumbing.ErrReferenceNotFound) {
		return fmt.Errorf("git backend: checkout branch: %w", err)
	}

	g.repo = repo
	g.worktree = worktree

	// Ensure sub-directories exist
	if err := os.MkdirAll(filepath.Join(g.options.Path, gitCredentialDir), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(g.options.Path, gitConfigDir), 0o755); err != nil {
		return err
	}

	// Attempt initial sync
	_ = g.pullLatest()
	return nil
}

// Close is a no-op for Git backend.
func (g *GitBackend) Close() error {
	return nil
}

// Health checks repository availability.
func (g *GitBackend) Health(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.pullLatest()
}

// Credential operations

func (g *GitBackend) GetCredential(ctx context.Context, id string) (map[string]interface{}, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.pullLatest(); err != nil {
		return nil, err
	}

	path := g.credentialPath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &ErrNotFound{Key: id}
		}
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (g *GitBackend) SetCredential(ctx context.Context, id string, data map[string]interface{}) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.pullLatest(); err != nil {
		return err
	}

	path := g.credentialPath(id)
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')

	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return err
	}
	if _, err := g.worktree.Add(relPath(g.options.Path, path)); err != nil {
		return err
	}
	if err := g.commit(fmt.Sprintf("Update credential %s", id)); err != nil {
		return err
	}
	return g.pushLatest()
}

func (g *GitBackend) DeleteCredential(ctx context.Context, id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.pullLatest(); err != nil {
		return err
	}

	path := g.credentialPath(id)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return &ErrNotFound{Key: id}
		}
		return err
	}
	if _, err := g.worktree.Remove(relPath(g.options.Path, path)); err != nil {
		return err
	}
	if err := g.commit(fmt.Sprintf("Delete credential %s", id)); err != nil {
		return err
	}
	return g.pushLatest()
}

func (g *GitBackend) ListCredentials(ctx context.Context) ([]string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.pullLatest(); err != nil {
		return nil, err
	}

	dir := filepath.Join(g.options.Path, gitCredentialDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		out = append(out, strings.TrimSuffix(entry.Name(), ".json"))
	}
	return out, nil
}

// Config operations

func (g *GitBackend) GetConfig(ctx context.Context, key string) (interface{}, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.pullLatest(); err != nil {
		return nil, err
	}

	path := g.configPath(key)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &ErrNotFound{Key: key}
		}
		return nil, err
	}

	var out interface{}
	if json.Valid(data) {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		if err := dec.Decode(&out); err != nil {
			return nil, err
		}
		return out, nil
	}
	return string(data), nil
}

func (g *GitBackend) SetConfig(ctx context.Context, key string, value interface{}) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.pullLatest(); err != nil {
		return err
	}

	path := g.configPath(key)
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}

	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	default:
		payload, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		data = append(payload, '\n')
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	if _, err := g.worktree.Add(relPath(g.options.Path, path)); err != nil {
		return err
	}
	if err := g.commit(fmt.Sprintf("Update config %s", key)); err != nil {
		return err
	}
	return g.pushLatest()
}

func (g *GitBackend) DeleteConfig(ctx context.Context, key string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.pullLatest(); err != nil {
		return err
	}

	path := g.configPath(key)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return &ErrNotFound{Key: key}
		}
		return err
	}
	if _, err := g.worktree.Remove(relPath(g.options.Path, path)); err != nil {
		return err
	}
	if err := g.commit(fmt.Sprintf("Delete config %s", key)); err != nil {
		return err
	}
	return g.pushLatest()
}

func (g *GitBackend) ListConfigs(ctx context.Context) (map[string]interface{}, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.pullLatest(); err != nil {
		return nil, err
	}

	dir := filepath.Join(g.options.Path, gitConfigDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make(map[string]interface{}, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		key := strings.TrimSuffix(entry.Name(), ".json")
		var value interface{}
		if json.Valid(data) {
			if err := json.Unmarshal(data, &value); err == nil {
				out[key] = value
				continue
			}
		}
		out[key] = string(data)
	}
	return out, nil
}

// Usage stats are not supported for git backend.
func (g *GitBackend) IncrementUsage(ctx context.Context, key string, field string, delta int64) error {
	return errGitUnsupported
}

func (g *GitBackend) GetUsage(ctx context.Context, key string) (map[string]interface{}, error) {
	return nil, errGitUnsupported
}

func (g *GitBackend) ResetUsage(ctx context.Context, key string) error {
	return errGitUnsupported
}

func (g *GitBackend) ListUsage(ctx context.Context) (map[string]map[string]interface{}, error) {
	return nil, errGitUnsupported
}

// Cache operations unsupported.
func (g *GitBackend) GetCache(ctx context.Context, key string) ([]byte, error) {
	return nil, errGitUnsupported
}
func (g *GitBackend) SetCache(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return errGitUnsupported
}
func (g *GitBackend) DeleteCache(ctx context.Context, key string) error {
	return errGitUnsupported
}

// Batch credential operations fall back to individual operations.
func (g *GitBackend) BatchGetCredentials(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	results := make(map[string]map[string]interface{}, len(ids))
	for _, id := range ids {
		data, err := g.GetCredential(ctx, id)
		if err != nil {
			return nil, err
		}
		results[id] = data
	}
	return results, nil
}

func (g *GitBackend) BatchSetCredentials(ctx context.Context, data map[string]map[string]interface{}) error {
	for id, payload := range data {
		if err := g.SetCredential(ctx, id, payload); err != nil {
			return err
		}
	}
	return nil
}

func (g *GitBackend) BatchDeleteCredentials(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := g.DeleteCredential(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// Transactions not supported.
func (g *GitBackend) BeginTransaction(ctx context.Context) (Transaction, error) {
	return nil, errGitUnsupported
}

// ExportData is not yet implemented for git backend.
func (g *GitBackend) ExportData(ctx context.Context) (map[string]interface{}, error) {
	return nil, errGitUnsupported
}

// ImportData is not yet implemented for git backend.
func (g *GitBackend) ImportData(ctx context.Context, data map[string]interface{}) error {
	return errGitUnsupported
}

// GetStorageStats returns basic information about the git repository.
func (g *GitBackend) GetStorageStats(ctx context.Context) (StorageStats, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	var lastCommit time.Time
	if g.repo != nil {
		head, err := g.repo.Head()
		if err == nil {
			if commit, err := g.repo.CommitObject(head.Hash()); err == nil {
				lastCommit = commit.Committer.When.UTC()
			}
		}
	}

	return StorageStats{
		Backend: "git",
		Healthy: true,
		Details: map[string]interface{}{
			"remote":       g.options.RemoteURL,
			"branch":       g.options.Branch,
			"path":         g.options.Path,
			"last_commit":  lastCommit,
			"credentials":  countFiles(filepath.Join(g.options.Path, gitCredentialDir)),
			"config_files": countFiles(filepath.Join(g.options.Path, gitConfigDir)),
		},
	}, nil
}

// Helper methods

func (g *GitBackend) credentialPath(id string) string {
	return filepath.Join(g.options.Path, gitCredentialDir, ensureJSONExt(id))
}

func (g *GitBackend) configPath(key string) string {
	return filepath.Join(g.options.Path, gitConfigDir, ensureJSONExt(key))
}

func (g *GitBackend) isExistingRepo() bool {
	_, err := os.Stat(filepath.Join(g.options.Path, ".git"))
	return err == nil
}

func (g *GitBackend) pullLatest() error {
	if g.repo == nil || g.worktree == nil || g.options.RemoteURL == "" {
		return nil
	}
	err := g.worktree.Pull(&git.PullOptions{
		RemoteName:    "origin",
		ReferenceName: plumbing.NewBranchReferenceName(g.options.Branch),
		SingleBranch:  true,
		Force:         false,
		Auth:          g.auth(),
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}
	return nil
}

func (g *GitBackend) pushLatest() error {
	if g.repo == nil || g.options.RemoteURL == "" {
		return nil
	}
	err := g.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       g.auth(),
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}
	return nil
}

func (g *GitBackend) commit(message string) error {
	if g.worktree == nil {
		return fmt.Errorf("git backend: worktree not initialised")
	}
	status, err := g.worktree.Status()
	if err != nil {
		return err
	}
	if status.IsClean() {
		return nil
	}
	_, err = g.worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  fallback(g.options.AuthorName, "gcli2api"),
			Email: fallback(g.options.AuthorEmail, "git@gcli2api.local"),
			When:  time.Now(),
		},
	})
	return err
}

func (g *GitBackend) auth() *http.BasicAuth {
	if g.options.Username == "" && g.options.Password == "" {
		return nil
	}
	return &http.BasicAuth{
		Username: g.options.Username,
		Password: g.options.Password,
	}
}

func ensureJSONExt(name string) string {
	if strings.HasSuffix(strings.ToLower(name), ".json") {
		return name
	}
	return name + ".json"
}

func relPath(base, target string) string {
	if rel, err := filepath.Rel(base, target); err == nil {
		return rel
	}
	return target
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if path == "~" {
			return home
		}
		if strings.HasPrefix(path, "~/") {
			return filepath.Join(home, path[2:])
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

func defaultBaseDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".gcli2api", "storage")
	}
	return "./storage"
}

func countFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			count++
		}
	}
	return count
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}
