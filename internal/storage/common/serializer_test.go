package common

import (
	"testing"
)

func TestSerializer_Marshal(t *testing.T) {
	s := NewSerializer()

	tests := []struct {
		name    string
		data    map[string]interface{}
		wantErr bool
	}{
		{
			name: "simple data",
			data: map[string]interface{}{
				"key": "value",
				"num": 123,
			},
			wantErr: false,
		},
		{
			name:    "nil data",
			data:    nil,
			wantErr: false,
		},
		{
			name:    "empty data",
			data:    map[string]interface{}{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := s.Marshal(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(payload) == 0 {
				t.Error("Marshal() returned empty payload")
			}
		})
	}
}

func TestSerializer_Unmarshal(t *testing.T) {
	s := NewSerializer()

	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid json",
			data:    []byte(`{"key":"value","num":123}`),
			wantErr: false,
		},
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: false,
		},
		{
			name:    "invalid json",
			data:    []byte(`{invalid}`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.Unmarshal(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("Unmarshal() returned nil result")
			}
		})
	}
}

func TestSerializer_CopyMap(t *testing.T) {
	s := NewSerializer()

	original := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	copy := s.CopyMap(original)

	// 修改副本不应影响原始数据
	copy["key1"] = "modified"

	if original["key1"] != "value1" {
		t.Error("CopyMap() did not create a proper copy")
	}
}

func TestSerializer_ValidateRequiredFields(t *testing.T) {
	s := NewSerializer()

	tests := []struct {
		name     string
		data     map[string]interface{}
		required []string
		wantErr  bool
	}{
		{
			name: "all fields present",
			data: map[string]interface{}{
				"field1": "value1",
				"field2": "value2",
			},
			required: []string{"field1", "field2"},
			wantErr:  false,
		},
		{
			name: "missing field",
			data: map[string]interface{}{
				"field1": "value1",
			},
			required: []string{"field1", "field2"},
			wantErr:  true,
		},
		{
			name:     "nil data",
			data:     nil,
			required: []string{"field1"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.ValidateRequiredFields(tt.data, tt.required)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequiredFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
