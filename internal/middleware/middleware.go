// Package middleware configures the global Gin middleware chain.
package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/piwriw/oas-go-template/internal/handler"
	"github.com/piwriw/oas-go-template/internal/logging"
)

const (
	corsMaxAge      = 12 * time.Hour
	requestIDHeader = "X-Request-ID"
)

// Options configures the global middleware chain.
type Options struct {
	ServiceName  string
	MaxBodyBytes int64
}

// Handlers returns the middleware chain in its required order.
func Handlers(opts Options, additional ...gin.HandlerFunc) []gin.HandlerFunc {
	handlers := []gin.HandlerFunc{
		handler.Recovery(),
		otelgin.Middleware(opts.ServiceName),
		logging.Middleware(),
		corsHandler(),
	}
	handlers = append(handlers, handler.BodyLimit(opts.MaxBodyBytes))
	return append(handlers, additional...)
}

// corsHandler allows browser clients from any origin without credentials.
func corsHandler() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", requestIDHeader},
		ExposeHeaders:    []string{requestIDHeader},
		AllowCredentials: false,
		MaxAge:           corsMaxAge,
	})
}

// Use installs the global middleware chain on r.
func Use(r *gin.Engine, opts Options, additional ...gin.HandlerFunc) {
	r.Use(Handlers(opts, additional...)...)
}
