#!/usr/bin/env bash
# 代码质量检查脚本
# 用于在提交前检查代码质量

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

print_header() {
    echo ""
    echo -e "${BLUE}=========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}=========================================${NC}"
    echo ""
}

# 检查 Go 代码格式
check_go_format() {
    print_info "Checking Go code format..."
    
    UNFORMATTED=$(gofmt -l . 2>/dev/null | grep -v vendor || true)
    
    if [ -n "$UNFORMATTED" ]; then
        print_error "The following Go files are not formatted:"
        echo "$UNFORMATTED"
        print_info "Run 'make fmt' to fix"
        return 1
    fi
    
    print_success "Go code format is correct"
    return 0
}

# 检查 Go lint
check_go_lint() {
    print_info "Checking Go lint..."
    
    # 使用 go vet
    if ! go vet ./... 2>&1; then
        print_error "Go vet found issues"
        return 1
    fi
    
    # 如果安装了 golangci-lint，使用它
    if command -v golangci-lint >/dev/null 2>&1; then
        if ! golangci-lint run 2>&1; then
            print_error "golangci-lint found issues"
            print_info "Run 'make lint-fix' to auto-fix some issues"
            return 1
        fi
    else
        print_warning "golangci-lint not installed, skipping advanced checks"
    fi
    
    print_success "Go lint checks passed"
    return 0
}

# 检查 TypeScript 类型
check_typescript() {
    print_info "Checking TypeScript types..."
    
    if [ ! -d "web" ]; then
        print_warning "web directory not found, skipping TypeScript check"
        return 0
    fi
    
    cd web
    if ! npm run typecheck 2>&1; then
        print_error "TypeScript type check failed"
        cd ..
        return 1
    fi
    cd ..
    
    print_success "TypeScript type check passed"
    return 0
}

# 检查前端 lint
check_frontend_lint() {
    print_info "Checking frontend lint..."
    
    if [ ! -d "web" ]; then
        print_warning "web directory not found, skipping frontend lint"
        return 0
    fi
    
    if ! npm run lint 2>&1; then
        print_error "Frontend lint found issues"
        print_info "Run 'make web-lint-fix' to auto-fix some issues"
        return 1
    fi
    
    print_success "Frontend lint checks passed"
    return 0
}

# 检查前端代码格式
check_frontend_format() {
    print_info "Checking frontend code format..."
    
    if [ ! -d "web" ]; then
        print_warning "web directory not found, skipping frontend format check"
        return 0
    fi
    
    if command -v prettier >/dev/null 2>&1; then
        if ! prettier --check "web/**/*.{js,ts,json,css,html}" 2>&1; then
            print_error "Frontend code format is incorrect"
            print_info "Run 'make web-fmt-fix' to fix"
            return 1
        fi
    else
        if ! npx prettier --check "web/**/*.{js,ts,json,css,html}" 2>&1; then
            print_error "Frontend code format is incorrect"
            print_info "Run 'make web-fmt-fix' to fix"
            return 1
        fi
    fi
    
    print_success "Frontend code format is correct"
    return 0
}

# 检查 Go 测试
check_go_tests() {
    print_info "Running Go tests..."
    
    if ! go test ./... -short 2>&1; then
        print_error "Go tests failed"
        return 1
    fi
    
    print_success "Go tests passed"
    return 0
}

# 检查前端测试
check_frontend_tests() {
    print_info "Running frontend tests..."
    
    if [ ! -d "web" ]; then
        print_warning "web directory not found, skipping frontend tests"
        return 0
    fi
    
    cd web
    if ! npm test -- --run 2>&1; then
        print_error "Frontend tests failed"
        cd ..
        return 1
    fi
    cd ..
    
    print_success "Frontend tests passed"
    return 0
}

# 检查依赖安全性
check_security() {
    print_info "Checking security vulnerabilities..."
    
    # Go 依赖检查
    if command -v govulncheck >/dev/null 2>&1; then
        if ! govulncheck ./... 2>&1; then
            print_warning "Go vulnerabilities found"
        else
            print_success "No Go vulnerabilities found"
        fi
    else
        print_warning "govulncheck not installed, skipping Go security check"
    fi
    
    # 前端依赖检查
    if [ -d "web" ]; then
        cd web
        if ! npm audit --audit-level=high 2>&1; then
            print_warning "Frontend vulnerabilities found"
            print_info "Run 'npm audit fix' to fix"
        else
            print_success "No frontend vulnerabilities found"
        fi
        cd ..
    fi
    
    return 0
}

# 主函数
main() {
    local exit_code=0
    local mode="${1:-all}"
    
    print_header "Code Quality Check"
    
    case "$mode" in
        format)
            check_go_format || exit_code=1
            check_frontend_format || exit_code=1
            ;;
        lint)
            check_go_lint || exit_code=1
            check_frontend_lint || exit_code=1
            ;;
        types)
            check_typescript || exit_code=1
            ;;
        test)
            check_go_tests || exit_code=1
            check_frontend_tests || exit_code=1
            ;;
        security)
            check_security || exit_code=1
            ;;
        quick)
            # 快速检查（格式 + lint + 类型）
            check_go_format || exit_code=1
            check_go_lint || exit_code=1
            check_typescript || exit_code=1
            check_frontend_lint || exit_code=1
            ;;
        all)
            # 完整检查
            check_go_format || exit_code=1
            check_frontend_format || exit_code=1
            check_go_lint || exit_code=1
            check_frontend_lint || exit_code=1
            check_typescript || exit_code=1
            check_go_tests || exit_code=1
            check_frontend_tests || exit_code=1
            check_security || exit_code=1
            ;;
        *)
            print_error "Unknown mode: $mode"
            echo "Usage: $0 [format|lint|types|test|security|quick|all]"
            exit 1
            ;;
    esac
    
    echo ""
    print_header "Summary"
    
    if [ $exit_code -eq 0 ]; then
        print_success "All checks passed! 🎉"
    else
        print_error "Some checks failed. Please fix the issues above."
    fi
    
    echo ""
    
    return $exit_code
}

# 运行主函数
main "$@"

