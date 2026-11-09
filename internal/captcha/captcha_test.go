package captcha

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTurnstileVerify(t *testing.T) {
	var gotSecret, gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSecret, gotToken = r.FormValue("secret"), r.FormValue("response")
		switch gotToken {
		case "good":
			w.Write([]byte(`{"success": true}`))
		case "garbled":
			w.Write([]byte(`not json`))
		default:
			w.Write([]byte(`{"success": false, "error-codes": ["invalid-input-response"]}`))
		}
	}))
	defer srv.Close()

	var o Opt
	o.Turnstile.Enabled, o.Turnstile.Key, o.Turnstile.Secret = true, "site", "shh"
	o.Login = true
	c := New(o)
	c.turnstile.URL = srv.URL

	if c.GetProvider() != ProviderTurnstile || c.FormField() != "cf-turnstile-response" || !c.OnLogin() {
		t.Fatalf("provider=%q field=%q login=%v", c.GetProvider(), c.FormField(), c.OnLogin())
	}
	if err, ok := c.Verify("good"); !ok || err != nil {
		t.Errorf("good token: ok=%v err=%v", ok, err)
	}
	if gotSecret != "shh" {
		t.Errorf("secret sent = %q", gotSecret)
	}
	if _, ok := c.Verify("bad"); ok {
		t.Error("rejected token passed")
	}
	if _, ok := c.Verify("garbled"); ok {
		t.Error("an unreadable answer passed; it must fail closed")
	}
}

func TestLoginNeedsAProvider(t *testing.T) {
	var o Opt
	o.Login = true
	if New(o).OnLogin() {
		t.Error("sign-in captcha on without any provider")
	}
}
