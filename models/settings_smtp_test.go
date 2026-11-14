package models

import (
	"encoding/json"
	"strings"
	"testing"
)

// A workspace's admins mustn't see anything of a platform-managed SMTP
// server's sign-in: the username is the provider's API token.
func TestManagedSMTPView_HidesServerDetails(t *testing.T) {
	s := Settings{SMTP: []SMTPServer{{
		Name: "email-postmark", UUID: "u1", Enabled: true, Host: "smtp.postmarkapp.com", Port: 587,
		AuthProtocol: "plain", Username: "server-token", Password: "server-token",
		FromAddresses: []string{"news@acme.example"}, ManagedByPlatform: true,
	}}}
	if !s.SMTPManaged() {
		t.Fatal("a platform-set server doesn't count as managed")
	}

	b, err := json.Marshal(s.ManagedSMTPView())
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"server-token", "smtp.postmarkapp.com", "587", "news@acme.example", "plain"} {
		if strings.Contains(string(b), leak) {
			t.Errorf("the managed view shows %q: %s", leak, b)
		}
	}
	v := s.ManagedSMTPView()[0]
	if v.Name != "email-postmark" || v.UUID != "u1" || !v.Enabled || !v.ManagedByPlatform {
		t.Errorf("view = %+v, want the name, ID, on state and managed flag kept", v)
	}
}

func TestSMTPManaged_OwnServers(t *testing.T) {
	s := Settings{SMTP: []SMTPServer{{Host: "smtp.example.com"}, {Host: "smtp2.example.com"}}}
	if s.SMTPManaged() {
		t.Error("a workspace's own servers count as managed")
	}
}
