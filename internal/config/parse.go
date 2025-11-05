package config

import "fmt"

// parsePort converts a numeric string into a TCP port (1-65535).
func parsePort(s string) (int, error) {
	var port int
	if _, err := fmt.Sscanf(s, "%d", &port); err != nil {
		return 0, err
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid port number: %d", port)
	}
	return port, nil
}

func parsePortOrDefault(s string) int {
	if p, err := parsePort(s); err == nil {
		return p
	}
	return 0
}

func parseInt(s string) (int, error) {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0, err
	}
	return n, nil
}
