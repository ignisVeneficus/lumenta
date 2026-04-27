package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TODO: move to config
type InputLimits struct {
	MaxURIBytes      int
	MaxPathBytes     int
	MaxSegments      int
	MaxQueryParams   int
	MaxParamValueLen int
	MaxBodyBytes     int64
}

func InputGuard() gin.HandlerFunc {
	// TODO: from config
	l := InputLimits{
		MaxURIBytes:      4096,
		MaxPathBytes:     2048,
		MaxSegments:      50,
		MaxQueryParams:   20,
		MaxParamValueLen: 512,
		MaxBodyBytes:     2 << 20, // 2MB
	}
	return func(c *gin.Context) {

		if len(c.Request.RequestURI) > l.MaxURIBytes {
			c.AbortWithStatusJSON(414, gin.H{"error": "URI too long"})
			return
		}

		path := c.Request.URL.Path
		if len(path) > l.MaxPathBytes {
			c.AbortWithStatusJSON(414, gin.H{"error": "Path too long"})
			return
		}

		segments := strings.Count(path, "/")
		if segments > l.MaxSegments {
			c.AbortWithStatusJSON(400, gin.H{"error": "Too many path segments"})
			return
		}

		query := c.Request.URL.Query()
		if len(query) > l.MaxQueryParams {
			c.AbortWithStatusJSON(400, gin.H{"error": "Too many query params"})
			return
		}

		for k, vals := range query {
			if len(k) > l.MaxParamValueLen {
				c.AbortWithStatusJSON(400, gin.H{"error": "Query key too long"})
				return
			}
			for _, v := range vals {
				if len(v) > l.MaxParamValueLen {
					c.AbortWithStatusJSON(400, gin.H{"error": "Query value too long"})
					return
				}
			}
		}

		if l.MaxBodyBytes > 0 && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, l.MaxBodyBytes)
		}

		c.Next()
	}
}
