package utils

import "testing"

func TestNormalizeBasePath(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "/"},
		{"/", "/"},
		{"api", "/api/"},
		{"/api", "/api/"},
		{"api/", "/api/"},
		{"/api/", "/api/"},
		{"//", "/"},
	}
	for _, c := range cases {
		if got := NormalizeBasePath(c.in); got != c.want {
			t.Fatalf("NormalizeBasePath(%q)=%q want %q", c.in, got, c.want)
		}
	}
}
