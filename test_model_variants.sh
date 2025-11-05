#!/bin/bash
set -e

echo "=== 测试模型变体功能 ==="
echo ""

# 测试 GenerateVariantsForModels 函数
cat > /tmp/test_variants.go << 'EOF'
package main

import (
	"fmt"
	"gcli2api-go/internal/models"
)

func main() {
	baseModels := []string{"gemini-2.5-pro", "gemini-2.5-flash"}
	
	fmt.Println("基础模型:")
	for _, m := range baseModels {
		fmt.Printf("  - %s\n", m)
	}
	fmt.Println()
	
	variants := models.GenerateVariantsForModels(baseModels)
	fmt.Printf("生成的变体总数: %d\n\n", len(variants))
	
	// 统计各类变体
	baseCount := 0
	fakeStreamCount := 0
	antiTruncCount := 0
	thinkingCount := 0
	searchCount := 0
	combinedCount := 0
	
	for _, v := range variants {
		features := models.ParseModelFeatures(v)
		
		if !features.FakeStreaming && !features.AntiTruncation && 
		   features.ThinkingLevel == "auto" && !features.Search {
			baseCount++
		} else if features.FakeStreaming && !features.AntiTruncation && 
		          features.ThinkingLevel == "auto" && !features.Search {
			fakeStreamCount++
		} else if !features.FakeStreaming && features.AntiTruncation && 
		          features.ThinkingLevel == "auto" && !features.Search {
			antiTruncCount++
		} else if !features.FakeStreaming && !features.AntiTruncation && 
		          features.ThinkingLevel != "auto" && !features.Search {
			thinkingCount++
		} else if !features.FakeStreaming && !features.AntiTruncation && 
		          features.ThinkingLevel == "auto" && features.Search {
			searchCount++
		} else {
			combinedCount++
		}
	}
	
	fmt.Println("变体统计:")
	fmt.Printf("  基础模型: %d\n", baseCount)
	fmt.Printf("  假流式前缀: %d\n", fakeStreamCount)
	fmt.Printf("  流式抗截断前缀: %d\n", antiTruncCount)
	fmt.Printf("  Thinking 后缀: %d\n", thinkingCount)
	fmt.Printf("  Search 后缀: %d\n", searchCount)
	fmt.Printf("  组合变体: %d\n", combinedCount)
	fmt.Println()
	
	// 显示示例
	fmt.Println("变体示例:")
	examples := []string{
		"gemini-2.5-pro",
		"假流式/gemini-2.5-pro",
		"流式抗截断/gemini-2.5-flash",
		"gemini-2.5-pro-maxthinking",
		"gemini-2.5-flash-nothinking",
		"gemini-2.5-pro-search",
		"假流式/gemini-2.5-pro-maxthinking",
		"流式抗截断/gemini-2.5-flash-search",
		"假流式/gemini-2.5-pro-maxthinking-search",
	}
	
	for _, ex := range examples {
		found := false
		for _, v := range variants {
			if v == ex {
				found = true
				break
			}
		}
		if found {
			fmt.Printf("  ✓ %s\n", ex)
		} else {
			fmt.Printf("  ✗ %s (未找到)\n", ex)
		}
	}
	
	fmt.Println("\n=== 测试完成 ===")
}
EOF

cd "$(dirname "$0")"
go run /tmp/test_variants.go

