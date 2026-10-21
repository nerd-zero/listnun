package auth

import (
	"testing"

	"github.com/nerd-zero/listnun/models"
)

func TestNeedsTwofaSetup(t *testing.T) {
	person := User{Type: UserTypeUser, PasswordLogin: true, TwofaType: "none"}
	if !NeedsTwofaSetup(person) {
		t.Error("password user without 2FA should be held")
	}
	withTOTP := person
	withTOTP.TwofaType = models.TwofaTypeTOTP
	if NeedsTwofaSetup(withTOTP) {
		t.Error("user with TOTP should pass")
	}
	sso := person
	sso.PasswordLogin = false
	if NeedsTwofaSetup(sso) {
		t.Error("single sign-on user should pass")
	}
	key := person
	key.APIKeyID = 3
	if NeedsTwofaSetup(key) {
		t.Error("API key request should pass")
	}
	api := person
	api.Type = UserTypeAPI
	if NeedsTwofaSetup(api) {
		t.Error("API user should pass")
	}
}

func TestTwofaSetupAllowed(t *testing.T) {
	allowed := [][2]string{
		{"GET", "/api/config"}, {"GET", "/api/profile"}, {"GET", "/api/lang/de"},
		{"POST", "/api/logout"}, {"GET", "/api/users/4/twofa/totp"}, {"PUT", "/api/users/4/twofa"},
	}
	for _, a := range allowed {
		if !TwofaSetupAllowed(a[0], a[1]) {
			t.Errorf("%s %s should be allowed", a[0], a[1])
		}
	}
	blocked := [][2]string{
		{"GET", "/api/subscribers"}, {"PUT", "/api/profile"}, {"DELETE", "/api/users/4/twofa"},
		{"PUT", "/api/settings"}, {"GET", "/api/users/4"},
	}
	for _, b := range blocked {
		if TwofaSetupAllowed(b[0], b[1]) {
			t.Errorf("%s %s should be blocked", b[0], b[1])
		}
	}
}
