package httpx

import "testing"

func TestJoinURL(t *testing.T) {
	cases := []struct {
		name string
		base string
		rel  string
		want string
	}{
		{"empty base, absolute rel", "", "https://example.com/foo", "https://example.com/foo"},
		{"empty base, relative rel", "", "/foo", "/foo"},
		{"base with trailing slash, rel with leading slash", "https://example.com/", "/foo", "https://example.com/foo"},
		{"base without trailing slash, rel without leading slash", "https://example.com", "foo", "https://example.com/foo"},
		{"base with trailing slash, rel without leading slash", "https://example.com/", "foo", "https://example.com/foo"},
		{"base path component", "https://example.com/api/v1", "/foo", "https://example.com/api/v1/foo"},
		{"absolute rel overrides base", "https://example.com/api", "https://other.com/bar", "https://other.com/bar"},
		{"both empty", "", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := joinURL(c.base, c.rel)
			if got != c.want {
				t.Errorf("joinURL(%q, %q) = %q, want %q", c.base, c.rel, got, c.want)
			}
		})
	}
}
