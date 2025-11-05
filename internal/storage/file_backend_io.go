package storage

import (
	"encoding/json"
	"os"
	"path/filepath"

	storagecommon "gcli2api-go/internal/storage/common"
)

// 从 file_backend.go 拆分：本地文件加载/保存辅助方法

func (f *FileBackend) loadAll() error {
	if err := f.loadCredentials(); err != nil {
		return err
	}
	if err := f.loadConfig(); err != nil {
		return err
	}
	if err := f.loadUsage(); err != nil {
		return err
	}
	return nil
}

func (f *FileBackend) loadCredentials() error {
	dir := filepath.Join(f.baseDir, "credentials")
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}
		id := file.Name()[:len(file.Name())-5]
		data, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			continue
		}
		cred := storagecommon.BorrowCredentialMap()
		if err := json.Unmarshal(data, &cred); err != nil {
			storagecommon.ReturnCredentialMap(cred)
			continue
		}
		f.replaceCredentialLocked(id, cred)
	}
	return nil
}

func (f *FileBackend) loadConfig() error {
	filePath := filepath.Join(f.baseDir, "config", "config.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &f.config)
}

func (f *FileBackend) loadUsage() error {
	dir := filepath.Join(f.baseDir, "usage")
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}
		key := file.Name()[:len(file.Name())-5]
		data, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			continue
		}
		var usage map[string]interface{}
		if err := json.Unmarshal(data, &usage); err != nil {
			continue
		}
		f.usage[key] = usage
	}
	return nil
}

func (f *FileBackend) saveAll() error {
	for id, cred := range f.credentials {
		if err := f.saveCredential(id, cred); err != nil {
			return err
		}
	}
	if err := f.saveConfig(); err != nil {
		return err
	}
	for key := range f.usage {
		if err := f.saveUsage(key); err != nil {
			return err
		}
	}
	return nil
}

func (f *FileBackend) saveCredential(id string, cred map[string]interface{}) error {
	data, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		return err
	}
	filePath := filepath.Join(f.baseDir, "credentials", id+".json")
	return os.WriteFile(filePath, data, 0600)
}

func (f *FileBackend) saveConfig() error {
	data, err := json.MarshalIndent(f.config, "", "  ")
	if err != nil {
		return err
	}
	filePath := filepath.Join(f.baseDir, "config", "config.json")
	return os.WriteFile(filePath, data, 0600)
}

func (f *FileBackend) saveUsage(key string) error {
	usage := f.usage[key]
	data, err := json.MarshalIndent(usage, "", "  ")
	if err != nil {
		return err
	}
	filePath := filepath.Join(f.baseDir, "usage", key+".json")
	return os.WriteFile(filePath, data, 0600)
}
