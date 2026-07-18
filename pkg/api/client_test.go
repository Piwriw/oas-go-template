package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetVersionWithResponseParsesTypedErrorStatuses(t *testing.T) {
	statuses := []int{
		http.StatusBadRequest,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusMethodNotAllowed,
		http.StatusRequestEntityTooLarge,
		http.StatusInternalServerError,
	}
	currentStatus := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(currentStatus)
		_, _ = fmt.Fprintf(w, `{"code":%d,"message":"request failed"}`, currentStatus)
	}))
	defer server.Close()

	client, err := NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatalf("NewClientWithResponses: %v", err)
	}

	for _, status := range statuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			currentStatus = status
			response, err := client.GetVersionWithResponse(t.Context())
			if err != nil {
				t.Fatalf("GetVersionWithResponse: %v", err)
			}
			if response.StatusCode() != status {
				t.Fatalf("status=%d, want %d", response.StatusCode(), status)
			}
			if got := typedErrorForStatus(response, status); got == nil {
				t.Fatalf("typed error for status %d is nil", status)
			} else if got.Message != "request failed" {
				t.Errorf("message=%q", got.Message)
			}
		})
	}
}

func typedErrorForStatus(response *GetVersionResponse, status int) *Error {
	switch status {
	case http.StatusBadRequest:
		return response.JSON400
	case http.StatusForbidden:
		return response.JSON403
	case http.StatusNotFound:
		return response.JSON404
	case http.StatusMethodNotAllowed:
		return response.JSON405
	case http.StatusRequestEntityTooLarge:
		return response.JSON413
	case http.StatusInternalServerError:
		return response.JSON500
	default:
		return nil
	}
}
