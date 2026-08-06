// Package middleware configures the global Gin middleware chain.
package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/piwriw/oas-go-template/internal/config"
	"github.com/piwriw/oas-go-template/internal/handler"
	"github.com/piwriw/oas-go-template/internal/logging"
)

// Options configures the global middleware chain.
type Options struct {
	ServiceName  string
	MaxBodyBytes int64
	CORS         config.CORSConfig
}

// Handlers returns the middleware chain in its required order.
func Handlers(opts Options, additional ...gin.HandlerFunc) []gin.HandlerFunc {
	handlers := []gin.HandlerFunc{
		handler.Recovery(),
		otelgin.Middleware(opts.ServiceName),
		logging.Middleware(),
	}
	if opts.CORS.Enabled {
		handlers = append(handlers, corsHandler(opts.CORS))
	}
	handlers = append(handlers, handler.BodyLimit(opts.MaxBodyBytes))
	return append(handlers, additional...)
}

// corsHandler translates validated cross-origin configuration into Gin middleware.
func corsHandler(cfg config.CORSConfig) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     cfg.AllowOrigins,
		AllowMethods:     cfg.AllowMethods,
		AllowHeaders:     cfg.AllowHeaders,
		ExposeHeaders:    cfg.ExposeHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
	})
}

// Use installs the global middleware chain on r.
func Use(r *gin.Engine, opts Options, additional ...gin.HandlerFunc) {
	r.Use(Handlers(opts, additional...)...)
}
