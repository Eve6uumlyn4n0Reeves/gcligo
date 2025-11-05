package common

import "sync"

var credentialMapPool = sync.Pool{
	New: func() any {
		return make(map[string]interface{}, 8)
	},
}

// BorrowCredentialMap retrieves a reusable map for credential data.
func BorrowCredentialMap() map[string]interface{} {
	return credentialMapPool.Get().(map[string]interface{})
}

// ReturnCredentialMap clears and returns a credential map to the pool.
func ReturnCredentialMap(m map[string]interface{}) {
	if m == nil {
		return
	}
	for k := range m {
		delete(m, k)
	}
	credentialMapPool.Put(m)
}

// CloneCredentialMap copies the provided data into a pooled map instance.
func CloneCredentialMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := BorrowCredentialMap()
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
