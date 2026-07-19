package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"

	"github.com/piwriw/oas-go-template/internal/oas"
)

func TestUseRunsAdditionalMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	called := false
	Use(r, Options{ServiceName: "middleware-test"}, func(c *gin.Context) {
		called = true
		c.Next()
	})
	r.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !called {
		t.Fatal("additional middleware was not called")
	}
}

func TestUseAddsDeprecationHeadersFromOpenAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spec := &openapi3.T{Paths: openapi3.NewPaths(
		openapi3.WithPath("/v1/orders/{id}", &openapi3.PathItem{Get: &openapi3.Operation{
			Deprecated: true,
			Extensions: map[string]any{
				oas.DeprecationDateExtension: "2026-08-01T00:00:00Z",
				oas.SunsetDateExtension:      "2027-02-01T00:00:00Z",
			},
		}}),
	)}
	r := gin.New()
	Use(r, Options{ServiceName: "middleware-test", OpenAPISpec: spec})
	r.GET("/v1/orders/:id", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/orders/123", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Deprecation"); got != "2026-08-01T00:00:00Z" {
		t.Errorf("Deprecation header=%q", got)
	}
	if got := rec.Header().Get("Sunset"); got != "2027-02-01T00:00:00Z" {
		t.Errorf("Sunset header=%q", got)
	}
}
