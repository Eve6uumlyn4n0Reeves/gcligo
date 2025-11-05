package common

// ShallowCopyMap creates a shallow copy of a map[string]interface{}.
func ShallowCopyMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// ShallowCopyNestedMap creates a shallow copy of a map[string]map[string]interface{}.
// Inner maps are copied using ShallowCopyMap to prevent accidental mutation.
func ShallowCopyNestedMap(src map[string]map[string]interface{}) map[string]map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = ShallowCopyMap(v)
	}
	return dst
}
