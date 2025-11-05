package config

import "golang.org/x/crypto/bcrypt"

// CheckManagementKey verifies whether the provided key matches the configured management credential.
func CheckManagementKey(cfg *Config, candidate string) bool {
	if cfg == nil || candidate == "" {
		return false
	}
	if cfg.ManagementKey != "" && candidate == cfg.ManagementKey {
		return true
	}
	if cfg.ManagementKeyHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(cfg.ManagementKeyHash), []byte(candidate)); err == nil {
			return true
		}
	}
	return false
}

// ManagementKeyValidator returns a closure suitable for middleware validation.
func ManagementKeyValidator(cfg *Config) func(string) bool {
	return func(candidate string) bool {
		return CheckManagementKey(cfg, candidate)
	}
}
