package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/errcode"
	"github.com/piwriw/oas-go-template/internal/logging"
)

// Stable public messages shared by the error responses below.
const (
	msgInvalidRequest      = "invalid request"
	msgRequestBodyTooLarge = "request body too large"
	msgRouteNotFound       = "route not found"
	msgInternalError       = "internal server error"
)

// StrictHandlerOptions maps generated binding failures to the stable public API error schema.
func StrictHandlerOptions() api.StrictGinServerOptions {
	return api.StrictGinServerOptions{
		RequestErrorHandlerFunc: func(c *gin.Context, err error) {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				writeError(c, http.StatusRequestEntityTooLarge, errcode.RequestBodyTooLarge, msgRequestBodyTooLarge, err)
				return
			}
			writeError(c, http.StatusBadRequest, errcode.InvalidRequest, msgInvalidRequest, err)
		},
		HandlerErrorFunc: func(c *gin.Context, err error) {
			writeError(c, http.StatusInternalServerError, errcode.Internal, msgInternalError, err)
		},
		ResponseErrorHandlerFunc: func(c *gin.Context, err error) {
			writeError(c, http.StatusInternalServerError, errcode.Internal, msgInternalError, err)
		},
	}
}

// Recovery converts handler panics into sanitized API error responses.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
		writeError(c, http.StatusInternalServerError, errcode.Internal, msgInternalError, fmt.Errorf("panic recovered: %v\n%s", recovered, debug.Stack()))
	})
}

// BodyLimit rejects oversized request bodies before contract binding and handler execution.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes <= 0 {
			c.Next()
			return
		}
		if c.Request.ContentLength > maxBytes {
			writeError(c, http.StatusRequestEntityTooLarge, errcode.RequestBodyTooLarge, msgRequestBodyTooLarge, fmt.Errorf("content length %d exceeds limit %d", c.Request.ContentLength, maxBytes))
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// OAPIValidationError sanitizes OpenAPI request validation failures for API callers.
func OAPIValidationError(c *gin.Context, message string, statusCode int) {
	code := errcode.InvalidRequest
	publicMessage := msgInvalidRequest
	if statusCode == http.StatusNotFound {
		code = errcode.NotFound
		publicMessage = msgRouteNotFound
	}
	writeError(c, statusCode, code, publicMessage, errors.New(message))
}

// NoRoute writes the common 404 response for paths outside the API contract.
func NoRoute(c *gin.Context) {
	writeError(c, http.StatusNotFound, errcode.NotFound, msgRouteNotFound, fmt.Errorf("%s %s", c.Request.Method, c.Request.URL.Path))
}

// NoMethod writes the common 405 response. Gin has already populated Allow.
func NoMethod(c *gin.Context) {
	writeError(c, http.StatusMethodNotAllowed, errcode.MethodNotAllowed, "method not allowed", fmt.Errorf("%s %s", c.Request.Method, c.Request.URL.Path))
}

// Forbidden writes a stable 403 response while retaining the detailed reason in logs.
func Forbidden(c *gin.Context, detail error) {
	writeError(c, http.StatusForbidden, errcode.Forbidden, "forbidden", detail)
}

// writeError logs private failure details and writes the sanitized public API error body.
func writeError(c *gin.Context, status int, code errcode.Code, message string, detail error) {
	logger := logging.From(c)
	args := []any{"status", status, "code", int32(code)}
	if detail != nil {
		args = append(args, "err", detail)
	}
	if status >= http.StatusInternalServerError {
		logger.ErrorContext(c.Request.Context(), "http error", args...)
	} else {
		logger.WarnContext(c.Request.Context(), "http error", args...)
	}

	if c.Writer.Written() {
		c.Abort()
		return
	}
	c.AbortWithStatusJSON(status, api.Error{Code: int32(code), Message: message})
}
