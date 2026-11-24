package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *Server) registerStaticRoutes(r *gin.Engine) {
	if s.assets == nil {
		return
	}

	serveIndex := func(c *gin.Context) {
		index, err := fs.ReadFile(s.assets, "index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "frontend is not built")
			return
		}
		index = injectAPIToken(index, s.apiToken)
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", index)
	}

	r.GET("/", serveIndex)

	r.NoRoute(func(c *gin.Context) {
		requestPath := c.Request.URL.Path
		if strings.HasPrefix(requestPath, "/api/") {
			apiError(c, http.StatusNotFound, "not_found", "API endpoint not found.")
			return
		}
		// Serve real files (JS/CSS/fonts), otherwise fall back to index.html so
		// client-side routing keeps working.
		filePath := strings.TrimPrefix(path.Clean(requestPath), "/")
		if filePath == "" || filePath == "." {
			serveIndex(c)
			return
		}
		info, err := fs.Stat(s.assets, filePath)
		if err != nil || info.IsDir() || filePath == "index.html" {
			serveIndex(c)
			return
		}
		if strings.HasPrefix(requestPath, "/assets/") {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			c.Header("Cache-Control", "public, max-age=3600")
		}
		c.FileFromFS(filePath, http.FS(s.assets))
	})
}

// injectAPIToken adds the per-process API token to the bootstrap HTML. The
// token is hex, so no HTML escaping is required.
func injectAPIToken(index []byte, token string) []byte {
	if token == "" {
		return index
	}
	script := []byte(`<script>window.__RESUME_TRAIL_TOKEN__="` + token + `";</script>`)
	if position := strings.Index(strings.ToLower(string(index)), "</head>"); position >= 0 {
		result := make([]byte, 0, len(index)+len(script))
		result = append(result, index[:position]...)
		result = append(result, script...)
		return append(result, index[position:]...)
	}
	result := make([]byte, 0, len(index)+len(script))
	result = append(result, script...)
	return append(result, index...)
}
