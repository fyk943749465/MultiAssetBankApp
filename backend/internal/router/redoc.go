package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const redocHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>go-chain API · ReDoc</title>
  <style>body { margin: 0; padding: 0; }</style>
</head>
<body>
  <redoc spec-url="/swagger/doc.json"></redoc>
  <script src="https://cdn.jsdelivr.net/npm/redoc@2.1.4/bundles/redoc.standalone.js"></script>
</body>
</html>`

// ReDoc serves a ReDoc UI that reads the same OpenAPI spec as Swagger UI (/swagger/doc.json).
func ReDoc(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(redocHTML))
}
