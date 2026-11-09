package auth

import (
	"strings"

	"github.com/nerd-zero/listnun/models"
)

// TwofaRequiredCode is the error code the API returns when a workspace
// requires two-step verification and the signed-in user hasn't set it up.
// The admin app sends the user to their profile when it sees it.
const TwofaRequiredCode = "twofa_required"

// NeedsTwofaSetup reports whether a workspace that requires two-step
// verification must hold this user back until they set it up. Only people
// signing in with a password are held: API keys and API users have no
// second factor, and single sign-on users rely on their identity provider.
func NeedsTwofaSetup(u User) bool {
	return u.APIKeyID == 0 && u.Type == UserTypeUser && u.PasswordLogin && u.TwofaType != models.TwofaTypeTOTP
}

// TwofaSetupAllowed lists the API calls a user may make while their
// two-step verification setup is still pending: load the app, read their
// own profile, set up TOTP and sign out.
func TwofaSetupAllowed(method, path string) bool {
	switch {
	case method == "GET" && (path == "/api/config" || path == "/api/profile" || path == "/api/health" ||
		strings.HasPrefix(path, "/api/lang/")):
		return true
	case method == "POST" && path == "/api/logout":
		return true
	case strings.HasPrefix(path, "/api/users/") && (strings.HasSuffix(path, "/twofa/totp") || strings.HasSuffix(path, "/twofa")):
		return method == "GET" || method == "PUT"
	}
	return false
}
