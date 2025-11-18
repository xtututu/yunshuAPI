package router

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func SetDocsRouter(router *gin.Engine) {
	// docs 主页路由
	router.GET("/docs", func(c *gin.Context) {
		c.File("/docs/index.html")
	})

	// docs 子页面路由
	router.GET("/docs/*filepath", func(c *gin.Context) {
		// 安全检查：防止路径遍历攻击
		requestedPath := c.Param("filepath")
		if filepath.Clean(requestedPath) != requestedPath {
			c.String(http.StatusBadRequest, "Invalid path")
			return
		}

		// 如果请求的是根目录或index.html，返回index.html
		if requestedPath == "/" || requestedPath == "/index.html" {
			c.File("/docs/index.html")
			return
		}

		// 尝试提供静态文件
		fullPath := "/docs" + requestedPath
		c.File(fullPath)
	})
}