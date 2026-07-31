package httpx

import "net/url"

// joinURL resolves an API request path beneath a configured base while preserving absolute overrides.
func joinURL(base, rel string) string {
	if base == "" {
		return rel
	}
	relURL, err := url.Parse(rel)
	if err != nil {
		return base + rel
	}
	if relURL.IsAbs() {
		return rel
	}
	joined, err := url.JoinPath(base, rel)
	if err != nil {
		return base + rel
	}
	return joined
}
