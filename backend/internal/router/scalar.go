package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// scalarHTML loads the same spec as Swagger/ReDoc (/swagger/doc.json). Same-origin, no proxy needed.
const scalarHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>go-chain API · Scalar</title>
  <style>
    body { margin: 0; height: 100vh; }
    #scalar-app { height: 100%; }
  </style>
</head>
<body>
  <div id="scalar-app"></div>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.28.5/dist/browser/standalone.js"></script>
  <script>
    Scalar.createApiReference('#scalar-app', { url: '/swagger/doc.json' });
  </script>
</body>
</html>`

// Scalar serves Scalar API Reference (OpenAPI/Swagger document from /swagger/doc.json).
func Scalar(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(scalarHTML))
}
