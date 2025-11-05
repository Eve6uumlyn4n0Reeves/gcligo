package common

import (
	"encoding/json"
	"fmt"
)

// Serializer 提供通用的序列化和反序列化功能
type Serializer struct{}

// NewSerializer 创建新的序列化器
func NewSerializer() *Serializer {
	return &Serializer{}
}

// Marshal 将 map 序列化为 JSON 字节数组
func (s *Serializer) Marshal(data map[string]interface{}) ([]byte, error) {
	if data == nil {
		return []byte("{}"), nil
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}
	return payload, nil
}

// Unmarshal 将 JSON 字节数组反序列化为 map
func (s *Serializer) Unmarshal(data []byte) (map[string]interface{}, error) {
	if len(data) == 0 {
		return make(map[string]interface{}), nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}
	return result, nil
}

// MarshalWithContext 带上下文信息的序列化（用于更好的错误消息）
func (s *Serializer) MarshalWithContext(data map[string]interface{}, context string) ([]byte, error) {
	payload, err := s.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal %s: %w", context, err)
	}
	return payload, nil
}

// UnmarshalWithContext 带上下文信息的反序列化（用于更好的错误消息）
func (s *Serializer) UnmarshalWithContext(data []byte, context string) (map[string]interface{}, error) {
	result, err := s.Unmarshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", context, err)
	}
	return result, nil
}

// CopyMap 深拷贝 map（避免数据竞争）
func (s *Serializer) CopyMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}

	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// MergeMap 合并两个 map（dst 中的值会被 src 覆盖）
func (s *Serializer) MergeMap(dst, src map[string]interface{}) map[string]interface{} {
	if dst == nil {
		dst = make(map[string]interface{})
	}
	if src == nil {
		return dst
	}

	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// FilterMap 根据键列表过滤 map
func (s *Serializer) FilterMap(src map[string]interface{}, keys []string) map[string]interface{} {
	if src == nil || len(keys) == 0 {
		return make(map[string]interface{})
	}

	keySet := make(map[string]bool, len(keys))
	for _, k := range keys {
		keySet[k] = true
	}

	dst := make(map[string]interface{})
	for k, v := range src {
		if keySet[k] {
			dst[k] = v
		}
	}
	return dst
}

// ValidateRequiredFields 验证必需字段是否存在
func (s *Serializer) ValidateRequiredFields(data map[string]interface{}, required []string) error {
	if data == nil {
		return fmt.Errorf("data is nil")
	}

	var missing []string
	for _, field := range required {
		if _, exists := data[field]; !exists {
			missing = append(missing, field)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %v", missing)
	}
	return nil
}
