// Package middleware configures the global Gin middleware chain.
package middleware

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/piwriw/oas-go-template/internal/config"
	"github.com/piwriw/oas-go-template/internal/handler"
	"github.com/piwriw/oas-go-template/internal/logging"
	"github.com/piwriw/oas-go-template/internal/oas"
)

// Options configures the global middleware chain.
type Options struct {
	ServiceName  string
	MaxBodyBytes int64
	CORS         config.CORSConfig
	OpenAPISpec  *openapi3.T
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
	if opts.OpenAPISpec != nil {
		handlers = append(handlers, deprecation(opts.OpenAPISpec))
	}
	return append(handlers, additional...)
}

func deprecation(spec *openapi3.T) gin.HandlerFunc {
	return func(c *gin.Context) {
		op := oas.FindOperation(spec, c.FullPath(), c.Request.Method)
		oas.ApplyDeprecationHeaders(c, op)
		c.Next()
	}
}

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
