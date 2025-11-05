package models

import "strings"

// Nano-banana 系列别名统一映射到 Gemini CLI 图片模型
// 需求确认：nano-banana -> gemini-2.5-flash-image-preview（不需要配置化）
const nanoBananaAliasTarget = "gemini-2.5-flash-image-preview"

// ResolveAlias 返回映射后的模型名；若无别名则原样返回且 ok=false。
func ResolveAlias(model string) (mapped string, ok bool) {
	m := strings.TrimSpace(strings.ToLower(model))
	if m == "nano-banana" || m == "nanobanana" || strings.HasPrefix(m, "nano-banana") {
		return nanoBananaAliasTarget, true
	}
	return model, false
}
