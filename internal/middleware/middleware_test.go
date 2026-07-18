package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
