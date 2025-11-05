package common

import (
	"encoding/json"
	"fmt"

	"gcli2api-go/internal/oauth"
)

// CredentialCodec centralizes JSON conversion for credential payloads across backends.
type CredentialCodec struct{}

// NewCredentialCodec creates a new codec instance.
func NewCredentialCodec() CredentialCodec {
	return CredentialCodec{}
}

// MarshalMap encodes a generic credential map into JSON bytes.
func (CredentialCodec) MarshalMap(data map[string]interface{}) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("credential map cannot be nil")
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("encode credential map: %w", err)
	}
	return payload, nil
}

// UnmarshalMap decodes JSON bytes into a map representation.
func (CredentialCodec) UnmarshalMap(payload []byte) (map[string]interface{}, error) {
	if len(payload) == 0 {
		return map[string]interface{}{}, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, fmt.Errorf("decode credential payload: %w", err)
	}
	return out, nil
}

// FromStruct converts oauth.Credentials into a map for API responses.
func (c CredentialCodec) FromStruct(cred *oauth.Credentials) (map[string]interface{}, error) {
	if cred == nil {
		return map[string]interface{}{}, nil
	}
	payload, err := json.Marshal(cred)
	if err != nil {
		return nil, fmt.Errorf("encode credential struct: %w", err)
	}
	return c.UnmarshalMap(payload)
}

// ToStruct converts a generic credential map into oauth.Credentials.
func (c CredentialCodec) ToStruct(data map[string]interface{}) (*oauth.Credentials, error) {
	if data == nil {
		return nil, fmt.Errorf("credential map cannot be nil")
	}
	payload, err := c.MarshalMap(data)
	if err != nil {
		return nil, err
	}
	var cred oauth.Credentials
	if err := json.Unmarshal(payload, &cred); err != nil {
		return nil, fmt.Errorf("decode credential json: %w", err)
	}
	return &cred, nil
}
