package server

import (
	"github.com/gin-gonic/gin"
)

// registerAdminStatic registers admin static asset routes (admin.js, admin.css)
// on any gin.IRoutes (either a group under basePath or the engine root for aliases).
// It mirrors the previous inline handlers and centralizes cache/MIME headers and path safety checks.
func registerAdminStatic(r gin.IRoutes) {
	// /admin.js
	r.GET("/admin.js", func(c *gin.Context) { serveAdminJS(c) })
	r.HEAD("/admin.js", func(c *gin.Context) { c.Header("Content-Type", "application/javascript"); c.Status(200) })

	// /admin.css
	r.GET("/admin.css", func(c *gin.Context) { serveAdminCSS(c) })
	r.HEAD("/admin.css", func(c *gin.Context) { c.Header("Content-Type", "text/css"); c.Status(200) })

}
