package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/errcode"
)

func decodeAPIError(t *testing.T, rec *httptest.ResponseRecorder) api.Error {
	t.Helper()
	var body api.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v; body=%s", err, rec.Body.String())
	}
	return body
}

func TestStrictServerOptionsSanitizesInternalErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	options := StrictServerOptions()
	r.GET("/", func(c *gin.Context) {
		options.HandlerErrorFunc(c, errors.New("database password leaked"))
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := decodeAPIError(t, rec)

	if rec.Code != http.StatusInternalServerError || body.Code != int32(errcode.Internal) {
		t.Fatalf("status=%d body=%+v", rec.Code, body)
	}
	if body.Message != "internal server error" || strings.Contains(rec.Body.String(), "password") {
		t.Errorf("internal detail leaked: %s", rec.Body.String())
	}
}

func TestRecoveryUsesSanitizedAPIError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery())
	r.GET("/", func(_ *gin.Context) {
		panic("sensitive panic detail")
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := decodeAPIError(t, rec)

	if rec.Code != http.StatusInternalServerError || body.Code != int32(errcode.Internal) {
		t.Fatalf("status=%d body=%+v", rec.Code, body)
	}
	if strings.Contains(rec.Body.String(), "sensitive") {
		t.Errorf("panic detail leaked: %s", rec.Body.String())
	}
}

func TestOAPIValidationErrorUsesStableResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		OAPIValidationError(c, "invalid value: secret", http.StatusBadRequest)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := decodeAPIError(t, rec)

	if rec.Code != http.StatusBadRequest || body.Code != int32(errcode.InvalidRequest) {
		t.Fatalf("status=%d body=%+v", rec.Code, body)
	}
	if body.Message != "invalid request" || strings.Contains(rec.Body.String(), "secret") {
		t.Errorf("validation detail leaked: %s", rec.Body.String())
	}
}
