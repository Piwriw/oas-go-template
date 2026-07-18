package handler

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

// StrictServerOptions replaces the generated handler's default {"msg": ...}
// errors with the public api.Error shape. Detailed errors are logged with the
// request ID and trace context, while callers receive stable messages only.
func StrictServerOptions() api.StrictGinServerOptions {
	return api.StrictGinServerOptions{
		RequestErrorHandlerFunc: func(c *gin.Context, err error) {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				writeError(c, http.StatusRequestEntityTooLarge, errcode.RequestBodyTooLarge, "request body too large", err)
				return
			}
			writeError(c, http.StatusBadRequest, errcode.InvalidRequest, "invalid request", err)
		},
		HandlerErrorFunc: func(c *gin.Context, err error) {
			writeError(c, http.StatusInternalServerError, errcode.Internal, "internal server error", err)
		},
		ResponseErrorHandlerFunc: func(c *gin.Context, err error) {
			writeError(c, http.StatusInternalServerError, errcode.Internal, "internal server error", err)
		},
	}
}

// Recovery returns a panic recovery middleware that keeps the public error
// shape consistent with handler and request parsing failures.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
		writeError(c, http.StatusInternalServerError, errcode.Internal, "internal server error", fmt.Errorf("panic recovered: %v\n%s", recovered, debug.Stack()))
	})
}

// BodyLimit caps request bodies before they reach a generated binder or handler.
// A non-positive limit disables this application-level check.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes <= 0 {
			c.Next()
			return
		}
		if c.Request.ContentLength > maxBytes {
			writeError(c, http.StatusRequestEntityTooLarge, errcode.RequestBodyTooLarge, "request body too large", fmt.Errorf("content length %d exceeds limit %d", c.Request.ContentLength, maxBytes))
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// OAPIValidationError adapts gin-middleware's callback to the common error
// response. The validator's detailed message remains in the log only.
func OAPIValidationError(c *gin.Context, message string, statusCode int) {
	code := errcode.InvalidRequest
	publicMessage := "invalid request"
	if statusCode == http.StatusNotFound {
		code = errcode.NotFound
		publicMessage = "route not found"
	}
	writeError(c, statusCode, code, publicMessage, errors.New(message))
}

// NoRoute writes the common 404 response for paths outside the API contract.
func NoRoute(c *gin.Context) {
	writeError(c, http.StatusNotFound, errcode.NotFound, "route not found", fmt.Errorf("%s %s", c.Request.Method, c.Request.URL.Path))
}

// NoMethod writes the common 405 response. Gin has already populated Allow.
func NoMethod(c *gin.Context) {
	writeError(c, http.StatusMethodNotAllowed, errcode.MethodNotAllowed, "method not allowed", fmt.Errorf("%s %s", c.Request.Method, c.Request.URL.Path))
}

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
