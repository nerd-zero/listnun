package tenant

import (
	"net/http/httptest"
	"testing"
)

func TestUnavailableRedirect(t *testing.T) {
	const console = "https://listnun.dev/"
	cases := []struct {
		name, method, path, status, console, want string
	}{
		{"suspended root", "GET", "/", "suspended", console, "https://listnun.dev/billing"},
		{"suspended admin", "GET", "/admin", "suspended", console, "https://listnun.dev/billing"},
		{"suspended admin page", "GET", "/admin/campaigns", "suspended", console, "https://listnun.dev/billing"},
		{"disabled admin", "GET", "/admin/login", "disabled", console, "https://listnun.dev"},
		{"no console configured", "GET", "/admin", "suspended", "", ""},
		{"api", "GET", "/api/lists", "suspended", console, ""},
		{"unsubscribe", "GET", "/subscription/abc/def", "suspended", console, ""},
		{"lookalike prefix", "GET", "/administrator", "suspended", console, ""},
		{"post", "POST", "/admin/login", "suspended", console, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(c.method, c.path, nil)
			if got := unavailableRedirect(r, c.status, c.console); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
