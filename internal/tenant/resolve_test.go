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

// The reply webhook serves every tenant at one URL; resolving a tenant
// from its Host would 404 it on an unknown host or 503 it whenever that
// host's tenant is suspended.
func TestIsCrossTenantPath(t *testing.T) {
	for path, want := range map[string]bool{
		"/api/operator/tenants":      true,
		"/webhooks/replies/postmark": true,
		"/webhooks/bounce":           false,
		"/webhooks/service/postmark": false,
		"/api/campaigns/1/replies":   false,
		"/admin/campaigns":           false,
	} {
		if got := isCrossTenantPath(path); got != want {
			t.Errorf("isCrossTenantPath(%q) = %v, want %v", path, got, want)
		}
	}
}
