package common

import (
	"fmt"
	"regexp"
	"strings"
)

// Validator 提供数据验证功能
type Validator struct {
	errorMapper *ErrorMapper
}

// NewValidator 创建新的验证器
func NewValidator() *Validator {
	return &Validator{
		errorMapper: NewErrorMapper(),
	}
}

// ValidationRule 验证规则
type ValidationRule struct {
	Field    string
	Required bool
	MinLen   int
	MaxLen   int
	Pattern  string
	Validate func(value interface{}) error
}

// ValidateID 验证 ID 格式
func (v *Validator) ValidateID(id string) error {
	if id == "" {
		return &ErrInvalidData{Reason: "ID cannot be empty"}
	}

	if len(id) > 255 {
		return &ErrInvalidData{Reason: "ID too long (max 255 characters)"}
	}

	// 检查是否包含非法字符
	if strings.ContainsAny(id, "\x00\n\r\t") {
		return &ErrInvalidData{Reason: "ID contains invalid characters"}
	}

	return nil
}

// ValidateCredentialID 验证凭证 ID 格式
func (v *Validator) ValidateCredentialID(id string) error {
	if err := v.ValidateID(id); err != nil {
		return err
	}

	// 凭证 ID 的额外验证规则
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(id) {
		return &ErrInvalidData{Reason: "credential ID must contain only alphanumeric characters, underscores, and hyphens"}
	}

	return nil
}

// ValidateConfigKey 验证配置键格式
func (v *Validator) ValidateConfigKey(key string) error {
	if err := v.ValidateID(key); err != nil {
		return err
	}

	// 配置键的额外验证规则
	if !regexp.MustCompile(`^[a-zA-Z0-9._-]+$`).MatchString(key) {
		return &ErrInvalidData{Reason: "config key must contain only alphanumeric characters, dots, underscores, and hyphens"}
	}

	return nil
}

// ValidateData 验证数据是否为空
func (v *Validator) ValidateData(data map[string]interface{}) error {
	if data == nil {
		return &ErrInvalidData{Reason: "data cannot be nil"}
	}

	if len(data) == 0 {
		return &ErrInvalidData{Reason: "data cannot be empty"}
	}

	return nil
}

// ValidateCredentialData 验证凭证数据
func (v *Validator) ValidateCredentialData(data map[string]interface{}) error {
	if err := v.ValidateData(data); err != nil {
		return err
	}

	// 检查必需字段
	requiredFields := []string{"id", "type"}
	for _, field := range requiredFields {
		if _, exists := data[field]; !exists {
			return &ErrInvalidData{Reason: fmt.Sprintf("missing required field: %s", field)}
		}
	}

	// 验证 type 字段
	credType, ok := data["type"].(string)
	if !ok {
		return &ErrInvalidData{Reason: "type must be a string"}
	}

	validTypes := []string{"oauth", "api_key", "service_account"}
	if !containsString(validTypes, credType) {
		return &ErrInvalidData{Reason: fmt.Sprintf("invalid credential type: %s", credType)}
	}

	return nil
}

// ValidateUsageData 验证使用统计数据
func (v *Validator) ValidateUsageData(data map[string]interface{}) error {
	if data == nil {
		return nil // 使用统计可以为空
	}

	// 验证数值字段
	numericFields := []string{"total_requests", "successful_requests", "failed_requests"}
	for _, field := range numericFields {
		if val, exists := data[field]; exists {
			switch v := val.(type) {
			case int, int32, int64, float32, float64:
				// 有效的数值类型
			default:
				return &ErrInvalidData{Reason: fmt.Sprintf("%s must be a number, got %T", field, v)}
			}
		}
	}

	return nil
}

// ValidateRules 根据规则验证数据
func (v *Validator) ValidateRules(data map[string]interface{}, rules []ValidationRule) error {
	for _, rule := range rules {
		value, exists := data[rule.Field]

		// 检查必需字段
		if rule.Required && !exists {
			return &ErrInvalidData{Reason: fmt.Sprintf("missing required field: %s", rule.Field)}
		}

		if !exists {
			continue // 可选字段不存在，跳过
		}

		// 字符串长度验证
		if strVal, ok := value.(string); ok {
			if rule.MinLen > 0 && len(strVal) < rule.MinLen {
				return &ErrInvalidData{
					Reason: fmt.Sprintf("%s too short (min %d characters)", rule.Field, rule.MinLen),
				}
			}
			if rule.MaxLen > 0 && len(strVal) > rule.MaxLen {
				return &ErrInvalidData{
					Reason: fmt.Sprintf("%s too long (max %d characters)", rule.Field, rule.MaxLen),
				}
			}

			// 正则表达式验证
			if rule.Pattern != "" {
				matched, err := regexp.MatchString(rule.Pattern, strVal)
				if err != nil {
					return fmt.Errorf("invalid pattern for %s: %w", rule.Field, err)
				}
				if !matched {
					return &ErrInvalidData{
						Reason: fmt.Sprintf("%s does not match required pattern", rule.Field),
					}
				}
			}
		}

		// 自定义验证函数
		if rule.Validate != nil {
			if err := rule.Validate(value); err != nil {
				return err
			}
		}
	}

	return nil
}

// SanitizeID 清理 ID（移除非法字符）
func (v *Validator) SanitizeID(id string) string {
	// 移除前后空格
	id = strings.TrimSpace(id)

	// 移除控制字符
	id = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, id)

	return id
}

// containsString 检查切片是否包含元素
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
