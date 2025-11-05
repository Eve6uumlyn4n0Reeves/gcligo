package config

import (
	"os"
	"strconv"
	"strings"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "true" || v == "1" || v == "yes" || v == "on"
}

func setIntFromEnv(key string, setter func(int)) {
	if v := getenv(key, ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			setter(n)
		}
	}
}

func setToggleFromEnv(key string, setter func(bool)) {
	v := strings.ToLower(strings.TrimSpace(getenv(key, "")))
	if v == "" {
		return
	}
	switch v {
	case "1", "true", "yes", "on":
		setter(true)
	case "0", "false", "no", "off":
		setter(false)
	}
}

func splitAndTrim(input, sep string) []string {
	parts := strings.Split(input, sep)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func normalizeBasePath(raw string) string {
	path := strings.TrimSpace(raw)
	if path == "" || path == "/" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	path = strings.TrimRight(path, "/")
	if path == "" {
		return ""
	}
	return path
}
