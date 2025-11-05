package server

import (
	"net/http"

	adminui "gcli2api-go/web"
	"github.com/gin-gonic/gin"
)

func serveEmbeddedFile(c *gin.Context, rel string, contentType string, cacheControl string) {
	if contentType != "" {
		c.Header("Content-Type", contentType)
	}
	if cacheControl != "" {
		c.Header("Cache-Control", cacheControl)
	}
	data, err := adminui.AssetsFS.ReadFile(rel)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Data(http.StatusOK, contentType, data)
}

func serveLoginHTML(c *gin.Context) {
	setNoCacheHeaders(c)
	serveEmbeddedFile(c, "login.html", "text/html; charset=utf-8", "")
}
func serveAdminHTML(c *gin.Context) {
	setNoCacheHeaders(c)
	serveEmbeddedFile(c, "admin.html", "text/html; charset=utf-8", "")
}
func serveAdminJS(c *gin.Context) {
	setNoCacheHeaders(c)
	serveEmbeddedFile(c, "admin.js", "application/javascript", "")
}
func serveAdminCSS(c *gin.Context) {
	setNoCacheHeaders(c)
	serveEmbeddedFile(c, "admin.css", "text/css", "")
}
