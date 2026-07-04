// Package middleware bundles project-wide gin middlewares.
package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger logs each request with method, path, status, latency.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("%s %s %d %s",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			time.Since(start),
		)
	}
}
