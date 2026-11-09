package core

import (
	"strings"
	"testing"

	"github.com/nerd-zero/listnun/models"
)

func TestValidAPIKeyName(t *testing.T) {
	for _, name := range []string{"checkout-svc", "crm.sync", "a1", "auth_service"} {
		if !ValidAPIKeyName(name) {
			t.Errorf("%q should be valid", name)
		}
	}
	for _, name := range []string{"", "a", "-leading", "Upper", "has space", "colon:name", strings.Repeat("a", 64)} {
		if ValidAPIKeyName(name) {
			t.Errorf("%q should be invalid", name)
		}
	}
}

func TestAPIKeyScopes(t *testing.T) {
	if !ValidAPIKeyScopes([]string{"tx:send", "subscribers:read"}) {
		t.Error("known scopes rejected")
	}
	if ValidAPIKeyScopes([]string{"tx:send", "settings:manage"}) {
		t.Error("a scope that isn't in the catalogue was accepted")
	}

	got := APIKeyPermissions([]string{"subscribers:read", "tx:send", "tx:send"})
	if strings.Join(got, ",") != "subscribers:get_all,tx:send" {
		t.Errorf("unexpected permissions %v", got)
	}
}

func TestNewAPIKeySecret(t *testing.T) {
	secret, hash, hint, err := newAPIKeySecret()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(secret, models.APIKeyPrefix) || len(secret) != len(models.APIKeyPrefix)+2*apiKeySecretBytes {
		t.Errorf("unexpected secret shape %q", secret)
	}
	if hint != secret[len(secret)-4:] {
		t.Errorf("hint %q isn't the secret's last four characters", hint)
	}
	if hash != hashAPIKeySecret(secret) || hash == secret {
		t.Error("hash doesn't match the secret or stores it in clear")
	}

	other, _, _, _ := newAPIKeySecret()
	if other == secret {
		t.Error("two secrets were the same")
	}
}
