package common

import "strings"

const (
	DoneMarker = "[DONE]"

	DoneInstruction = `严格执行以下输出结束规则：

1. 当你完成完整回答时，必须在输出的最后单独一行输出：[DONE]
2. [DONE] 标记表示你的回答已经完全结束，这是必需的结束标记
3. 只有输出了 [DONE] 标记，系统才认为你的回答是完整的
4. 如果你的回答被截断，系统会要求你继续输出剩余内容
5. 无论回答长短，都必须以 [DONE] 标记结束

示例格式：
你的回答内容...
更多回答内容...
[DONE]

注意：请确保 [DONE] 必须单独占一行，前面不要有任何其他字符。`

	ContinuationPrompt = `请从刚才被截断的地方继续输出剩余的所有内容。

重要提醒：
1. 不要重复前面已经输出的内容
2. 直接继续输出，无需任何前言或解释
3. 当你完整完成所有内容输出后，必须在最后一行单独输出：[DONE]
4. [DONE] 标记表示你的回答已经完全结束，这是必需的结束标记

现在请继续输出：`
)

var doneMarkerLower = strings.ToLower(DoneMarker)

// EqualDoneMarker returns true when the provided value equals the done marker ignoring case and surrounding whitespace.
func EqualDoneMarker(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), DoneMarker)
}

// HasDoneMarker reports whether the done marker appears in the provided text ignoring case.
func HasDoneMarker(text string) bool {
	if text == "" {
		return false
	}
	return strings.Contains(strings.ToLower(text), doneMarkerLower)
}

// StripDoneMarker removes standalone done marker lines from the text, comparing case-insensitively.
func StripDoneMarker(text string) string {
	if text == "" {
		return text
	}
	lines := strings.Split(text, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if EqualDoneMarker(line) {
			continue
		}
		kept = append(kept, line)
	}
	cleaned := strings.Join(kept, "\n")
	return strings.TrimSpace(cleaned)
}
