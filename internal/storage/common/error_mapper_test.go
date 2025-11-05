package common

import (
	"context"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestErrorMapper_MapRedisError(t *testing.T) {
	em := NewErrorMapper()

	tests := []struct {
		name     string
		err      error
		key      string
		wantType string
		wantNil  bool
	}{
		{
			name:    "nil error",
			err:     nil,
			key:     "test",
			wantNil: true,
		},
		{
			name:     "redis.Nil",
			err:      redis.Nil,
			key:      "test",
			wantType: "not_found",
		},
		{
			name:     "context.Canceled",
			err:      context.Canceled,
			key:      "test",
			wantType: "canceled",
		},
		{
			name:     "context.DeadlineExceeded",
			err:      context.DeadlineExceeded,
			key:      "test",
			wantType: "timeout",
		},
		{
			name:     "other error",
			err:      errors.New("some error"),
			key:      "test",
			wantType: "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := em.MapRedisError(tt.err, tt.key)

			if tt.wantNil {
				if result != nil {
					t.Errorf("MapRedisError() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Error("MapRedisError() returned nil, want error")
				return
			}

			switch tt.wantType {
			case "not_found":
				if !em.IsNotFound(result) {
					t.Errorf("MapRedisError() should return NotFound error, got %v", result)
				}
			case "canceled", "timeout", "other":
				if em.IsNotFound(result) {
					t.Errorf("MapRedisError() should not return NotFound error, got %v", result)
				}
			}
		})
	}
}

func TestErrorMapper_MapMongoError(t *testing.T) {
	em := NewErrorMapper()

	tests := []struct {
		name     string
		err      error
		key      string
		wantType string
		wantNil  bool
	}{
		{
			name:    "nil error",
			err:     nil,
			key:     "test",
			wantNil: true,
		},
		{
			name:     "mongo.ErrNoDocuments",
			err:      mongo.ErrNoDocuments,
			key:      "test",
			wantType: "not_found",
		},
		{
			name:     "context.Canceled",
			err:      context.Canceled,
			key:      "test",
			wantType: "canceled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := em.MapMongoError(tt.err, tt.key)

			if tt.wantNil {
				if result != nil {
					t.Errorf("MapMongoError() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Error("MapMongoError() returned nil, want error")
				return
			}

			if tt.wantType == "not_found" && !em.IsNotFound(result) {
				t.Errorf("MapMongoError() should return NotFound error, got %v", result)
			}
		})
	}
}

func TestErrorMapper_IsNotFound(t *testing.T) {
	em := NewErrorMapper()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "NotFound error",
			err:  &ErrNotFound{Key: "test"},
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("some error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := em.IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorMapper_IsAlreadyExists(t *testing.T) {
	em := NewErrorMapper()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "AlreadyExists error",
			err:  &ErrAlreadyExists{Key: "test"},
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("some error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := em.IsAlreadyExists(tt.err); got != tt.want {
				t.Errorf("IsAlreadyExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorMapper_WrapError(t *testing.T) {
	em := NewErrorMapper()

	tests := []struct {
		name      string
		err       error
		operation string
		resource  string
		wantNil   bool
		wantMsg   string
	}{
		{
			name:    "nil error",
			err:     nil,
			wantNil: true,
		},
		{
			name:      "wrap error",
			err:       errors.New("original error"),
			operation: "get",
			resource:  "credential",
			wantMsg:   "get credential",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := em.WrapError(tt.err, tt.operation, tt.resource)

			if tt.wantNil {
				if result != nil {
					t.Errorf("WrapError() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Error("WrapError() returned nil, want error")
				return
			}

			if tt.wantMsg != "" && !contains(result.Error(), tt.wantMsg) {
				t.Errorf("WrapError() error message = %v, want to contain %v", result.Error(), tt.wantMsg)
			}
		})
	}
}

func TestErrNotFound_Error(t *testing.T) {
	err := &ErrNotFound{Key: "test-key"}
	expected := "not found: test-key"

	if err.Error() != expected {
		t.Errorf("ErrNotFound.Error() = %v, want %v", err.Error(), expected)
	}
}

func TestErrAlreadyExists_Error(t *testing.T) {
	err := &ErrAlreadyExists{Key: "test-key"}
	expected := "already exists: test-key"

	if err.Error() != expected {
		t.Errorf("ErrAlreadyExists.Error() = %v, want %v", err.Error(), expected)
	}
}

func TestErrInvalidData_Error(t *testing.T) {
	err := &ErrInvalidData{Reason: "invalid format"}
	expected := "invalid data: invalid format"

	if err.Error() != expected {
		t.Errorf("ErrInvalidData.Error() = %v, want %v", err.Error(), expected)
	}
}
