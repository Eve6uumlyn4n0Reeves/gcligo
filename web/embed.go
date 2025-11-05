package adminui

import "embed"

// AssetsFS 嵌入管理控制台静态资源（HTML / CSS / JS 模块）。
//
//go:embed admin.html admin.css login.html dist/**
var AssetsFS embed.FS
