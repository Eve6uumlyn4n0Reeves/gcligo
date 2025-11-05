package credential

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// FileSource 从本地目录加载/保存凭证，兼容旧版基于 authDir 的实现。
type FileSource struct {
	dir  string
	name string
}

// NewFileSource 构造文件来源。dir 应使用绝对路径或提前展开 ~。
func NewFileSource(dir string) *FileSource {
	clean := filepath.Clean(dir)
	return &FileSource{
		dir:  clean,
		name: "file:" + clean,
	}
}

// Dir 返回当前目录。
func (s *FileSource) Dir() string {
	return s.dir
}

func (s *FileSource) Name() string {
	return s.name
}

// Load 实现 CredentialSource 接口，读取目录内的 JSON 凭证文件。
func (s *FileSource) Load(_ context.Context) ([]*Credential, error) {
	if s.dir == "" {
		return nil, fmt.Errorf("file source directory not configured")
	}
	files, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("read credential directory: %w", err)
	}
	var creds []*Credential
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}
		if strings.HasSuffix(strings.ToLower(file.Name()), ".state.json") {
			continue
		}
		fullPath := filepath.Join(s.dir, file.Name())
		data, err := os.ReadFile(fullPath)
		if err != nil {
			log.WithError(err).Warnf("credential file source: failed to read %s", file.Name())
			continue
		}
		var cred Credential
		if err := json.Unmarshal(data, &cred); err != nil {
			log.WithError(err).Warnf("credential file source: failed to parse %s", file.Name())
			continue
		}
		if cred.ID == "" {
			cred.ID = file.Name()
		}
		if cred.AccessToken != "" || cred.RefreshToken != "" {
			cred.Type = "oauth"
		} else if cred.APIKey != "" {
			cred.Type = "api_key"
		}
		cred.Source = s.Name()
		creds = append(creds, &cred)
	}
	return creds, nil
}

// Save 将凭证写回目录。
func (s *FileSource) Save(_ context.Context, cred *Credential) error {
	if s.dir == "" {
		return fmt.Errorf("file source directory not configured")
	}
	if cred == nil {
		return fmt.Errorf("credential is nil")
	}
	if cred.ID == "" {
		return fmt.Errorf("credential id is required")
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("prepare credential directory: %w", err)
	}
	path := filepath.Join(s.dir, ensureJSONExtension(cred.ID))
	data, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credential %s: %w", cred.ID, err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write credential %s: %w", cred.ID, err)
	}
	return nil
}

// Delete 移除凭证文件。
func (s *FileSource) Delete(_ context.Context, id string) error {
	if s.dir == "" {
		return fmt.Errorf("file source directory not configured")
	}
	if id == "" {
		return fmt.Errorf("credential id is required")
	}
	path := filepath.Join(s.dir, ensureJSONExtension(id))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete credential %s: %w", id, err)
	}
	return nil
}

// RestoreState 从磁盘恢复凭证状态。
func (s *FileSource) RestoreState(_ context.Context, cred *Credential) error {
	if cred == nil {
		return nil
	}
	path := s.statePath(cred.ID)
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read state file: %w", err)
	}
	var state CredentialState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("parse state file: %w", err)
	}
	cred.RestoreState(&state)
	return nil
}

// PersistState 写入状态文件。
func (s *FileSource) PersistState(_ context.Context, cred *Credential, state *CredentialState) error {
	if cred == nil || state == nil {
		return nil
	}
	if s.dir == "" {
		return fmt.Errorf("file source directory not configured")
	}
	if cred.ID == "" {
		return fmt.Errorf("credential id missing")
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("prepare state directory: %w", err)
	}
	path := s.statePath(cred.ID)
	if path == "" {
		return fmt.Errorf("state path unavailable")
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write temp state: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename state: %w", err)
	}
	return nil
}

// DeleteState 移除状态文件。
func (s *FileSource) DeleteState(_ context.Context, id string) error {
	if id == "" {
		return nil
	}
	path := s.statePath(id)
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete state: %w", err)
	}
	return nil
}

func (s *FileSource) statePath(id string) string {
	if s.dir == "" || id == "" {
		return ""
	}
	base := strings.TrimSuffix(id, filepath.Ext(id))
	return filepath.Join(s.dir, base+credentialStateSuffix)
}

func ensureJSONExtension(id string) string {
	if filepath.Ext(id) != "" {
		return id
	}
	return id + ".json"
}
