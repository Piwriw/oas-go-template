// Package middleware builds the process-wide Gin middleware chain.
//
// The order of the built-in handlers is intentional: recovery must wrap the
// whole request, otelgin must run before logging so trace context is available
// to log records, and the body limit must run before request binding.
package middleware

import (
	"fmt"
	"strings"

	ginCors "github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/piwriw/oas-go-template/internal/config"
	"github.com/piwriw/oas-go-template/internal/handler"
	"github.com/piwriw/oas-go-template/internal/logging"
)

// Options contains the settings needed to build the global middleware chain.
// Additional handlers passed to Handlers or Use run after the built-in chain,
// which is the default-safe extension point for application middleware.
type Options struct {
	ServiceName  string
	MaxBodyBytes int64
	CORS         config.CORSConfig
}

// Handlers returns the global middleware chain in its required order.
// Additional handlers are appended after the built-in handlers. Use Gin route
// groups for middleware that should only apply to a subset of endpoints.
func Handlers(opts Options, additional ...gin.HandlerFunc) []gin.HandlerFunc {
	handlers := []gin.HandlerFunc{
		handler.Recovery(),
		otelgin.Middleware(opts.ServiceName),
		logging.Middleware(),
	}
	if opts.CORS.Enabled {
		handlers = append(handlers, cors(opts.CORS))
	}
	handlers = append(handlers, handler.BodyLimit(opts.MaxBodyBytes))
	return append(handlers, additional...)
}

func cors(cfg config.CORSConfig) gin.HandlerFunc {
	corsConfig := ginCors.Config{
		AllowMethods:     cfg.AllowMethods,
		AllowHeaders:     cfg.AllowHeaders,
		ExposeHeaders:    cfg.ExposeHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
	}
	if allowsAllOrigins(cfg.AllowOrigins) {
		corsConfig.AllowAllOrigins = true
	} else {
		corsConfig.AllowOriginWithContextFunc = func(_ *gin.Context, origin string) bool {
			return allowsOrigin(cfg.AllowOrigins, origin)
		}
	}
	delegate := ginCors.New(corsConfig)
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin != "" && !sameOrigin(c, origin) && !allowsOrigin(cfg.AllowOrigins, origin) {
			handler.Forbidden(c, fmt.Errorf("cors origin %q is not allowed", origin))
			return
		}
		delegate(c)
	}
}

func allowsAllOrigins(origins []string) bool {
	for _, origin := range origins {
		if strings.TrimSpace(origin) == "*" {
			return true
		}
	}
	return false
}

func allowsOrigin(origins []string, origin string) bool {
	for _, allowed := range origins {
		allowed = strings.TrimSpace(allowed)
		if allowed == "*" || strings.EqualFold(allowed, origin) {
			return true
		}
	}
	return false
}

func sameOrigin(c *gin.Context, origin string) bool {
	host := c.Request.Host
	return origin == "http://"+host || origin == "https://"+host
}

// Use installs the global middleware chain on r. Additional handlers are
// appended after the built-in chain; for route-specific handlers, use r.Group
// or a registered route's middleware instead.
func Use(r *gin.Engine, opts Options, additional ...gin.HandlerFunc) {
	r.Use(Handlers(opts, additional...)...)
}
