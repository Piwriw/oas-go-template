// Package oas contains checks and runtime helpers for the OpenAPI contract.
package oas

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// OpenAPI extension names used by the versioning and deprecation policy.
const (
	APIVersionExtension      = "x-api-version"
	VersioningExtension      = "x-versioning"
	DeprecationDateExtension = "x-deprecation-date"
	SunsetDateExtension      = "x-sunset-date"
	URLPrefixStrategy        = "url-prefix"
	deprecationHeader        = "Deprecation"
	sunsetHeader             = "Sunset"
)

var (
	apiVersionPattern  = regexp.MustCompile(`^v[0-9]+$`)
	versionPathPattern = regexp.MustCompile(`^/v[0-9]+(?:/|$)`)
)

// Validate checks the repository's API versioning and deprecation conventions.
// The document remains the source of truth, so this check is intentionally
// independent of generated server or client code.
func Validate(doc *openapi3.T) error {
	if doc == nil {
		return fmt.Errorf("OpenAPI document is nil")
	}

	apiVersion, ok := stringExtension(doc.Extensions, APIVersionExtension)
	if !ok || !apiVersionPattern.MatchString(apiVersion) {
		return fmt.Errorf("%s must be a version like v1", APIVersionExtension)
	}

	versioning, ok := mapExtension(doc.Extensions, VersioningExtension)
	if !ok {
		return fmt.Errorf("%s must be an object", VersioningExtension)
	}
	strategy, ok := stringExtension(versioning, "strategy")
	if !ok || strategy != URLPrefixStrategy {
		return fmt.Errorf("%s.strategy must be %q", VersioningExtension, URLPrefixStrategy)
	}
	exceptions, err := stringSliceExtension(versioning, "unversioned_paths")
	if err != nil {
		return fmt.Errorf("%s.unversioned_paths: %w", VersioningExtension, err)
	}
	exceptionSet := make(map[string]struct{}, len(exceptions))
	for _, path := range exceptions {
		if path == "" || path[0] != '/' {
			return fmt.Errorf("%s.unversioned_paths contains invalid path %q", VersioningExtension, path)
		}
		exceptionSet[path] = struct{}{}
	}

	if doc.Paths != nil {
		for _, path := range doc.Paths.Keys() {
			if _, exempt := exceptionSet[path]; !exempt && !versionPathPattern.MatchString(path) {
				return fmt.Errorf("path %q must use a /vN URL prefix", path)
			}
		}
	}

	return validateDeprecations(doc)
}

// ValidateDeprecations checks only operation deprecation metadata. It is
// exported separately for tools that load a document without using the URL
// versioning policy.
func ValidateDeprecations(doc *openapi3.T) error {
	return validateDeprecations(doc)
}

func validateDeprecations(doc *openapi3.T) error {
	if doc == nil || doc.Paths == nil {
		return nil
	}

	paths := doc.Paths.Keys()
	sort.Strings(paths)
	for _, path := range paths {
		item := doc.Paths.Value(path)
		if item == nil {
			continue
		}
		methods := item.Operations()
		methodNames := make([]string, 0, len(methods))
		for method := range methods {
			methodNames = append(methodNames, method)
		}
		sort.Strings(methodNames)
		for _, method := range methodNames {
			op := methods[method]
			if op == nil || !op.Deprecated {
				continue
			}
			deprecatedAt, err := timeExtension(op.Extensions, DeprecationDateExtension)
			if err != nil {
				return fmt.Errorf("%s %s: %w", method, path, err)
			}
			sunsetAt, err := timeExtension(op.Extensions, SunsetDateExtension)
			if err != nil {
				return fmt.Errorf("%s %s: %w", method, path, err)
			}
			if !sunsetAt.After(deprecatedAt) {
				return fmt.Errorf("%s %s: %s must be after %s", method, path, SunsetDateExtension, DeprecationDateExtension)
			}
		}
	}
	return nil
}

func stringExtension(extensions map[string]any, name string) (string, bool) {
	value, ok := extensions[name]
	if !ok {
		return "", false
	}
	stringValue, ok := value.(string)
	return strings.TrimSpace(stringValue), ok && strings.TrimSpace(stringValue) != ""
}

func mapExtension(extensions map[string]any, name string) (map[string]any, bool) {
	value, ok := extensions[name]
	if !ok {
		return nil, false
	}
	object, ok := value.(map[string]any)
	return object, ok
}

func stringSliceExtension(extensions map[string]any, name string) ([]string, error) {
	value, ok := extensions[name]
	if !ok {
		return nil, fmt.Errorf("must be a list")
	}
	switch values := value.(type) {
	case []string:
		return values, nil
	case []any:
		result := make([]string, 0, len(values))
		for _, value := range values {
			item, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("contains non-string value %v", value)
			}
			result = append(result, item)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("must be a list")
	}
}

func timeExtension(extensions map[string]any, name string) (time.Time, error) {
	value, ok := stringExtension(extensions, name)
	if !ok {
		return time.Time{}, fmt.Errorf("deprecated operation requires %s as an RFC3339 timestamp", name)
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be an RFC3339 timestamp: %w", name, err)
	}
	return parsed, nil
}

// FindOperation resolves a Gin route template to an OpenAPI operation.
func FindOperation(doc *openapi3.T, ginPath, method string) *openapi3.Operation {
	if doc == nil || doc.Paths == nil {
		return nil
	}
	item := doc.Paths.Find(ginPathToOASPath(ginPath))
	if item == nil {
		return nil
	}
	return item.Operations()[strings.ToUpper(method)]
}

// ApplyDeprecationHeaders adds the dates declared on a deprecated operation.
// The values intentionally remain RFC3339 so clients can compare them with
// the OAS metadata without lossy HTTP-date conversion.
func ApplyDeprecationHeaders(c interface{ Header(string, string) }, op *openapi3.Operation) {
	if op == nil || !op.Deprecated {
		return
	}
	if deprecatedAt, err := timeExtension(op.Extensions, DeprecationDateExtension); err == nil {
		c.Header(deprecationHeader, deprecatedAt.Format(time.RFC3339))
	}
	if sunsetAt, err := timeExtension(op.Extensions, SunsetDateExtension); err == nil {
		c.Header(sunsetHeader, sunsetAt.Format(time.RFC3339))
	}
}

func ginPathToOASPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") || strings.HasPrefix(part, "*") {
			parts[i] = "{" + part[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}

// DeprecatedHeaderNames documents the response headers emitted for deprecated
// operations and is useful when configuring CORS exposed headers.
func DeprecatedHeaderNames() []string {
	return []string{http.CanonicalHeaderKey(deprecationHeader), http.CanonicalHeaderKey(sunsetHeader)}
}
