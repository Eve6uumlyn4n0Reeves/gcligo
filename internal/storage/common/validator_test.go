package common

import (
	"errors"
	"testing"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Fatal("Expected non-nil validator")
	}
	if v.errorMapper == nil {
		t.Error("Expected error mapper to be initialized")
	}
}

func TestValidateID(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"Valid ID", "test-id-123", false},
		{"Empty ID", "", true},
		{"ID with newline", "test\nid", true},
		{"ID with null byte", "test\x00id", true},
		{"ID with tab", "test\tid", true},
		{"ID with carriage return", "test\rid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Test length limits separately
	t.Run("Too long ID", func(t *testing.T) {
		longID := ""
		for i := 0; i < 256; i++ {
			longID += "a"
		}
		err := v.ValidateID(longID)
		if err == nil {
			t.Error("Expected error for ID longer than 255 characters")
		}
	})

	t.Run("Valid 255 char ID", func(t *testing.T) {
		longID := ""
		for i := 0; i < 255; i++ {
			longID += "a"
		}
		err := v.ValidateID(longID)
		if err != nil {
			t.Errorf("Expected no error for 255 char ID, got %v", err)
		}
	})
}

func TestValidateCredentialID(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"Valid credential ID", "cred-123", false},
		{"Valid with underscore", "cred_123", false},
		{"Valid alphanumeric", "abc123XYZ", false},
		{"Empty ID", "", true},
		{"With spaces", "cred 123", true},
		{"With dots", "cred.123", true},
		{"With special chars", "cred@123", true},
		{"With slash", "cred/123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateCredentialID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentialID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfigKey(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"Valid config key", "app.config.key", false},
		{"Valid with hyphen", "app-config-key", false},
		{"Valid with underscore", "app_config_key", false},
		{"Valid alphanumeric", "config123", false},
		{"Empty key", "", true},
		{"With spaces", "app config", true},
		{"With special chars", "app@config", true},
		{"With slash", "app/config", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateConfigKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfigKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateData(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		data    map[string]interface{}
		wantErr bool
	}{
		{"Valid data", map[string]interface{}{"key": "value"}, false},
		{"Nil data", nil, true},
		{"Empty data", map[string]interface{}{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCredentialData(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		data    map[string]interface{}
		wantErr bool
	}{
		{
			"Valid oauth credential",
			map[string]interface{}{"id": "cred1", "type": "oauth"},
			false,
		},
		{
			"Valid api_key credential",
			map[string]interface{}{"id": "cred2", "type": "api_key"},
			false,
		},
		{
			"Valid service_account credential",
			map[string]interface{}{"id": "cred3", "type": "service_account"},
			false,
		},
		{
			"Missing id",
			map[string]interface{}{"type": "oauth"},
			true,
		},
		{
			"Missing type",
			map[string]interface{}{"id": "cred1"},
			true,
		},
		{
			"Invalid type",
			map[string]interface{}{"id": "cred1", "type": "invalid"},
			true,
		},
		{
			"Type not string",
			map[string]interface{}{"id": "cred1", "type": 123},
			true,
		},
		{
			"Nil data",
			nil,
			true,
		},
		{
			"Empty data",
			map[string]interface{}{},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateCredentialData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentialData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUsageData(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		data    map[string]interface{}
		wantErr bool
	}{
		{"Nil data is valid", nil, false},
		{"Empty data is valid", map[string]interface{}{}, false},
		{
			"Valid numeric fields",
			map[string]interface{}{
				"total_requests":      100,
				"successful_requests": 90,
				"failed_requests":     10,
			},
			false,
		},
		{
			"Valid with float",
			map[string]interface{}{"total_requests": 100.5},
			false,
		},
		{
			"Invalid string value",
			map[string]interface{}{"total_requests": "invalid"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateUsageData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsageData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRules(t *testing.T) {
	v := NewValidator()

	t.Run("Required field missing", func(t *testing.T) {
		rules := []ValidationRule{
			{Field: "name", Required: true},
		}
		data := map[string]interface{}{}

		err := v.ValidateRules(data, rules)
		if err == nil {
			t.Error("Expected error for missing required field")
		}
	})

	t.Run("MinLen validation", func(t *testing.T) {
		rules := []ValidationRule{
			{Field: "name", MinLen: 5},
		}
		data := map[string]interface{}{"name": "abc"}

		err := v.ValidateRules(data, rules)
		if err == nil {
			t.Error("Expected error for string too short")
		}
	})

	t.Run("MaxLen validation", func(t *testing.T) {
		rules := []ValidationRule{
			{Field: "name", MaxLen: 5},
		}
		data := map[string]interface{}{"name": "abcdefgh"}

		err := v.ValidateRules(data, rules)
		if err == nil {
			t.Error("Expected error for string too long")
		}
	})

	t.Run("Pattern validation", func(t *testing.T) {
		rules := []ValidationRule{
			{Field: "email", Pattern: `^[a-z]+@[a-z]+\.[a-z]+$`},
		}
		data := map[string]interface{}{"email": "invalid-email"}

		err := v.ValidateRules(data, rules)
		if err == nil {
			t.Error("Expected error for pattern mismatch")
		}
	})

	t.Run("Custom validation", func(t *testing.T) {
		rules := []ValidationRule{
			{
				Field: "age",
				Validate: func(value interface{}) error {
					age, ok := value.(int)
					if !ok {
						return errors.New("age must be int")
					}
					if age < 0 {
						return errors.New("age must be positive")
					}
					return nil
				},
			},
		}
		data := map[string]interface{}{"age": -5}

		err := v.ValidateRules(data, rules)
		if err == nil {
			t.Error("Expected error from custom validation")
		}
	})

	t.Run("All validations pass", func(t *testing.T) {
		rules := []ValidationRule{
			{Field: "name", Required: true, MinLen: 3, MaxLen: 10},
			{Field: "email", Pattern: `^[a-z]+@[a-z]+\.[a-z]+$`},
		}
		data := map[string]interface{}{
			"name":  "john",
			"email": "john@example.com",
		}

		err := v.ValidateRules(data, rules)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

func TestSanitizeID(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"No changes needed", "test-id", "test-id"},
		{"Trim spaces", "  test-id  ", "test-id"},
		{"Remove control chars", "test\x00\x01id", "testid"},
		{"Remove newline", "test\nid", "testid"},
		{"Remove tab", "test\tid", "testid"},
		{"Remove DEL char", "test\x7Fid", "testid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.SanitizeID(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeID() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"Found", []string{"a", "b", "c"}, "b", true},
		{"Not found", []string{"a", "b", "c"}, "d", false},
		{"Empty slice", []string{}, "a", false},
		{"Empty item", []string{"a", "b"}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsString(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("containsString() = %v, want %v", result, tt.expected)
			}
		})
	}
}
