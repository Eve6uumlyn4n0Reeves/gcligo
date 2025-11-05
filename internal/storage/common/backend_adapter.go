package common

import (
	"encoding/json"
	"fmt"

	"gcli2api-go/internal/oauth"
)

// BackendAdapter centralizes common JSON (de)serialization helpers shared by
// storage backends. It avoids duplicating serializer/codec wiring in each
// backend implementation.
type BackendAdapter struct {
	serializer      *Serializer
	credentialCodec CredentialCodec
}

// NewBackendAdapter constructs a helper with default serializer/codec.
func NewBackendAdapter() BackendAdapter {
	return BackendAdapter{
		serializer:      NewSerializer(),
		credentialCodec: NewCredentialCodec(),
	}
}

// MarshalCredential encodes a credential map to JSON bytes.
func (a BackendAdapter) MarshalCredential(data map[string]interface{}) ([]byte, error) {
	return a.credentialCodec.MarshalMap(data)
}

// UnmarshalCredential decodes JSON bytes into a credential map.
func (a BackendAdapter) UnmarshalCredential(payload []byte) (map[string]interface{}, error) {
	return a.credentialCodec.UnmarshalMap(payload)
}

// BatchUnmarshalCredentials decodes several payloads into maps keyed by credential ID.
func (a BackendAdapter) BatchUnmarshalCredentials(raw map[string][]byte) (map[string]map[string]interface{}, error) {
	result := make(map[string]map[string]interface{}, len(raw))
	for id, payload := range raw {
		if len(payload) == 0 {
			continue
		}
		decoded, err := a.UnmarshalCredential(payload)
		if err != nil {
			return nil, fmt.Errorf("decode credential %s: %w", id, err)
		}
		result[id] = decoded
	}
	return result, nil
}

// BatchMarshalCredentials encodes a map of credential data into byte payloads.
func (a BackendAdapter) BatchMarshalCredentials(data map[string]map[string]interface{}) (map[string][]byte, error) {
	result := make(map[string][]byte, len(data))
	for id, credData := range data {
		payload, err := a.MarshalCredential(credData)
		if err != nil {
			return nil, fmt.Errorf("encode credential %s: %w", id, err)
		}
		result[id] = payload
	}
	return result, nil
}

// CredentialFromStruct encodes oauth credentials into a map representation.
func (a BackendAdapter) CredentialFromStruct(cred *oauth.Credentials) (map[string]interface{}, error) {
	return a.credentialCodec.FromStruct(cred)
}

// CredentialToStruct decodes a generic map into oauth credentials.
func (a BackendAdapter) CredentialToStruct(data map[string]interface{}) (*oauth.Credentials, error) {
	return a.credentialCodec.ToStruct(data)
}

// BatchCredentialsToStruct converts generic maps to oauth credentials.
func (a BackendAdapter) BatchCredentialsToStruct(data map[string]map[string]interface{}) (map[string]*oauth.Credentials, error) {
	result := make(map[string]*oauth.Credentials, len(data))
	for id, credData := range data {
		cred, err := a.CredentialToStruct(credData)
		if err != nil {
			return nil, fmt.Errorf("decode credential %s: %w", id, err)
		}
		result[id] = cred
	}
	return result, nil
}

// BatchCredentialsFromStruct converts oauth credentials to generic maps keyed by id.
func (a BackendAdapter) BatchCredentialsFromStruct(data map[string]*oauth.Credentials) (map[string]map[string]interface{}, error) {
	result := make(map[string]map[string]interface{}, len(data))
	for id, cred := range data {
		mapped, err := a.CredentialFromStruct(cred)
		if err != nil {
			return nil, fmt.Errorf("encode credential %s: %w", id, err)
		}
		result[id] = mapped
	}
	return result, nil
}

// MarshalDocument encodes a generic document (config/cache/etc.).
func (a BackendAdapter) MarshalDocument(context string, data map[string]interface{}) ([]byte, error) {
	return a.serializer.MarshalWithContext(data, context)
}

// UnmarshalDocument decodes a generic document payload.
func (a BackendAdapter) UnmarshalDocument(context string, payload []byte) (map[string]interface{}, error) {
	return a.serializer.UnmarshalWithContext(payload, context)
}

// CopyMap returns a shallow copy, useful before mutating cached data.
func (a BackendAdapter) CopyMap(src map[string]interface{}) map[string]interface{} {
	return a.serializer.CopyMap(src)
}

// MarshalValue serializes any value (struct/map/string/etc.) into JSON.
func (a BackendAdapter) MarshalValue(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

// UnmarshalValue parses JSON into a generic interface{}.
func (a BackendAdapter) UnmarshalValue(payload []byte) (interface{}, error) {
	var out interface{}
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, err
	}
	return out, nil
}
