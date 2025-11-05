#!/usr/bin/env bash
# 覆盖率检查脚本
# 用于检查 Go 和前端测试覆盖率是否达到阈值

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认阈值
GO_THRESHOLD=${GO_THRESHOLD:-50}
WEB_THRESHOLD=${WEB_THRESHOLD:-40}

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 检查 Go 覆盖率
check_go_coverage() {
    print_info "Checking Go test coverage..."
    
    # 运行测试并生成覆盖率报告
    if ! go test ./... -coverprofile=coverage.out 2>&1 | tee test_output.txt; then
        print_error "Go tests failed"
        rm -f coverage.out test_output.txt
        return 1
    fi
    
    # 提取覆盖率
    if [ ! -f coverage.out ]; then
        print_error "Coverage file not generated"
        rm -f test_output.txt
        return 1
    fi
    
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    
    # 清理临时文件
    rm -f coverage.out test_output.txt
    
    # 检查阈值
    if [ -z "$COVERAGE" ]; then
        print_error "Failed to extract coverage percentage"
        return 1
    fi
    
    print_info "Go coverage: ${COVERAGE}% (threshold: ${GO_THRESHOLD}%)"
    
    # 使用 bc 进行浮点数比较
    if command -v bc >/dev/null 2>&1; then
        if [ "$(echo "$COVERAGE < $GO_THRESHOLD" | bc -l)" -eq 1 ]; then
            print_error "Go coverage ${COVERAGE}% is below threshold ${GO_THRESHOLD}%"
            return 1
        fi
    else
        # 如果没有 bc，使用整数比较
        COVERAGE_INT=${COVERAGE%.*}
        if [ "$COVERAGE_INT" -lt "$GO_THRESHOLD" ]; then
            print_error "Go coverage ${COVERAGE}% is below threshold ${GO_THRESHOLD}%"
            return 1
        fi
    fi
    
    print_success "Go coverage ${COVERAGE}% meets threshold ${GO_THRESHOLD}%"
    return 0
}

# 检查前端覆盖率
check_web_coverage() {
    print_info "Checking frontend test coverage..."
    
    # 检查是否存在 web 目录
    if [ ! -d "web" ]; then
        print_warning "web directory not found, skipping frontend coverage check"
        return 0
    fi
    
    # 运行前端测试
    cd web
    if ! npm run test:coverage 2>&1 | tee ../web_test_output.txt; then
        print_error "Frontend tests failed"
        cd ..
        rm -f web_test_output.txt
        return 1
    fi
    cd ..
    
    # 检查覆盖率摘要文件
    if [ ! -f "web/coverage/coverage-summary.json" ]; then
        print_warning "Coverage summary not found, checking test output..."
        
        # 尝试从测试输出中提取覆盖率
        if grep -q "All files" web_test_output.txt; then
            COVERAGE=$(grep "All files" web_test_output.txt | awk '{print $4}' | sed 's/%//' || echo "0")
        else
            print_warning "Could not extract coverage from test output"
            rm -f web_test_output.txt
            return 0
        fi
    else
        # 从 JSON 文件中提取覆盖率
        if command -v jq >/dev/null 2>&1; then
            COVERAGE=$(jq -r '.total.lines.pct' web/coverage/coverage-summary.json)
        else
            # 如果没有 jq，使用 grep 和 sed
            COVERAGE=$(grep -o '"lines":{"total":[0-9]*,"covered":[0-9]*,"skipped":[0-9]*,"pct":[0-9.]*' web/coverage/coverage-summary.json | grep -o 'pct":[0-9.]*' | cut -d':' -f2 || echo "0")
        fi
    fi
    
    rm -f web_test_output.txt
    
    # 检查阈值
    if [ -z "$COVERAGE" ] || [ "$COVERAGE" = "null" ]; then
        print_warning "Failed to extract frontend coverage percentage"
        return 0
    fi
    
    print_info "Frontend coverage: ${COVERAGE}% (threshold: ${WEB_THRESHOLD}%)"
    
    # 使用 bc 进行浮点数比较
    if command -v bc >/dev/null 2>&1; then
        if [ "$(echo "$COVERAGE < $WEB_THRESHOLD" | bc -l)" -eq 1 ]; then
            print_error "Frontend coverage ${COVERAGE}% is below threshold ${WEB_THRESHOLD}%"
            return 1
        fi
    else
        # 如果没有 bc，使用整数比较
        COVERAGE_INT=${COVERAGE%.*}
        if [ "$COVERAGE_INT" -lt "$WEB_THRESHOLD" ]; then
            print_error "Frontend coverage ${COVERAGE}% is below threshold ${WEB_THRESHOLD}%"
            return 1
        fi
    fi
    
    print_success "Frontend coverage ${COVERAGE}% meets threshold ${WEB_THRESHOLD}%"
    return 0
}

# 主函数
main() {
    local exit_code=0
    
    echo ""
    print_info "========================================="
    print_info "Coverage Threshold Check"
    print_info "========================================="
    echo ""
    
    # 检查 Go 覆盖率
    if ! check_go_coverage; then
        exit_code=1
    fi
    
    echo ""
    
    # 检查前端覆盖率
    if ! check_web_coverage; then
        exit_code=1
    fi
    
    echo ""
    print_info "========================================="
    
    if [ $exit_code -eq 0 ]; then
        print_success "All coverage checks passed!"
    else
        print_error "Some coverage checks failed"
    fi
    
    echo ""
    
    return $exit_code
}

# 运行主函数
main "$@"

