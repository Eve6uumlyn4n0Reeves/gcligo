package constants

import "os"

// Version information (injected at build time)
var (
	// Version 应用版本号（通过 -ldflags 注入）
	Version = "dev"

	// BuildTime 构建时间（通过 -ldflags 注入）
	BuildTime = "unknown"

	// GitCommit Git 提交哈希（通过 -ldflags 注入）
	GitCommit = "unknown"

	// GoVersion Go 版本
	GoVersion = "unknown"
)

// Admin Asset Versions
const (
	// AdminAssetVersionDefault 管理资产默认版本
	AdminAssetVersionDefault = "20251026"
)

// GetAdminAssetVersion 获取管理资产版本（支持环境变量覆盖）
func GetAdminAssetVersion() string {
	if v := os.Getenv("ADMIN_ASSET_VERSION"); v != "" {
		return v
	}
	return AdminAssetVersionDefault
}

// GetVersion 获取应用版本信息
func GetVersion() string {
	return Version
}

// GetBuildTime 获取构建时间
func GetBuildTime() string {
	return BuildTime
}

// GetGitCommit 获取 Git 提交哈希
func GetGitCommit() string {
	return GitCommit
}

// GetGoVersion 获取 Go 版本
func GetGoVersion() string {
	return GoVersion
}

// GetFullVersion 获取完整版本信息
func GetFullVersion() string {
	return Version + " (" + GitCommit + ") built at " + BuildTime + " with " + GoVersion
}
