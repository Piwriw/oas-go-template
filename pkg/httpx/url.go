package httpx

import "net/url"

// joinURL resolves rel against base using "API client" semantics: base
// acts as a URL prefix, rel is path-joined under it.
//
// If base is empty, rel is returned unchanged. If rel is itself absolute
// (has a scheme), it overrides base entirely. Otherwise base + rel are
// path-joined via url.JoinPath, which normalizes slash boundaries and
// preserves the base's path components (so "https://x/api/v1" + "/foo"
// yields "https://x/api/v1/foo", not "https://x/foo").
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
