package config

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestCheckManagementKeyPlain(t *testing.T) {
	cfg := &Config{ManagementKey: "secret"}
	if !CheckManagementKey(cfg, "secret") {
		t.Fatalf("expected plain key to validate")
	}
	if CheckManagementKey(cfg, "other") {
		t.Fatalf("unexpected match for wrong key")
	}
}

func TestCheckManagementKeyHash(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	cfg := &Config{ManagementKeyHash: string(hash)}
	if !CheckManagementKey(cfg, "secret") {
		t.Fatalf("expected hashed key to validate")
	}
	if CheckManagementKey(cfg, "other") {
		t.Fatalf("unexpected hash match for wrong key")
	}
}

func TestManagementKeyValidator(t *testing.T) {
	cfg := &Config{ManagementKey: "plain"}
	validate := ManagementKeyValidator(cfg)
	if !validate("plain") {
		t.Fatalf("expected validator success")
	}
	if validate("wrong") {
		t.Fatalf("expected validator failure")
	}
}
