package oas

import (
	"net/http"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestValidateVersioningPolicy(t *testing.T) {
	doc := testDocument()
	if err := Validate(doc); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	doc.Paths.Set("/orders", &openapi3.PathItem{Get: openapi3.NewOperation()})
	if err := Validate(doc); err == nil {
		t.Fatal("Validate() accepted an unversioned business path")
	}
}

func TestValidateDeprecationMetadata(t *testing.T) {
	doc := testDocument()
	op := doc.Paths.Find("/v1/orders/{orderID}").Get
	op.Deprecated = true
	op.Extensions = map[string]any{
		DeprecationDateExtension: "2026-08-01T00:00:00Z",
		SunsetDateExtension:      "2027-02-01T00:00:00Z",
	}
	if err := ValidateDeprecations(doc); err != nil {
		t.Fatalf("ValidateDeprecations() error = %v", err)
	}

	tests := []struct {
		name       string
		deprecated string
		sunset     string
	}{
		{name: "missing deprecated date", sunset: "2027-02-01T00:00:00Z"},
		{name: "invalid sunset date", deprecated: "2026-08-01T00:00:00Z", sunset: "tomorrow"},
		{name: "sunset before deprecated", deprecated: "2027-02-01T00:00:00Z", sunset: "2026-08-01T00:00:00Z"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			op.Extensions = map[string]any{
				DeprecationDateExtension: test.deprecated,
				SunsetDateExtension:      test.sunset,
			}
			if err := ValidateDeprecations(doc); err == nil {
				t.Fatal("ValidateDeprecations() accepted invalid metadata")
			}
		})
	}
}

func TestFindOperationAndApplyDeprecationHeaders(t *testing.T) {
	doc := testDocument()
	op := FindOperation(doc, "/v1/orders/:orderID", http.MethodGet)
	if op == nil {
		t.Fatal("FindOperation() returned nil")
	}
	op.Deprecated = true
	op.Extensions = map[string]any{
		DeprecationDateExtension: "2026-08-01T00:00:00Z",
		SunsetDateExtension:      "2027-02-01T00:00:00Z",
	}

	headers := headerCapture{}
	ApplyDeprecationHeaders(headers, op)
	if got := headers[DeprecationDateExtension]; got != "" {
		t.Fatalf("unexpected extension key in headers: %q", got)
	}
	if got := headers["Deprecation"]; got != "2026-08-01T00:00:00Z" {
		t.Errorf("Deprecation header = %q", got)
	}
	if got := headers["Sunset"]; got != "2027-02-01T00:00:00Z" {
		t.Errorf("Sunset header = %q", got)
	}
}

type headerCapture map[string]string

func (h headerCapture) Header(name, value string) {
	h[name] = value
}

func testDocument() *openapi3.T {
	return &openapi3.T{
		Extensions: map[string]any{
			APIVersionExtension: "v1",
			VersioningExtension: map[string]any{
				"strategy":          URLPrefixStrategy,
				"unversioned_paths": []string{"/healthz", "/readyz", "/version"},
			},
		},
		Paths: openapi3.NewPaths(
			openapi3.WithPath("/healthz", &openapi3.PathItem{Get: openapi3.NewOperation()}),
			openapi3.WithPath("/v1/orders/{orderID}", &openapi3.PathItem{Get: openapi3.NewOperation()}),
		),
	}
}
