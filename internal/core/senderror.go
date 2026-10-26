package core

import (
	"strings"
)

// DescribeSendError turns a messenger's send error into a line for the
// workspace's activity log, leading with what most likely went wrong so
// someone without server access knows where to look. The raw error stays
// at the end for whoever needs the detail.
func DescribeSendError(err error) string {
	raw := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(err.Error()), ":"))
	lower := strings.ToLower(raw)

	switch {
	case strings.HasPrefix(raw, "535") || strings.Contains(lower, "authentication failed") ||
		strings.Contains(lower, "username and password not accepted"):
		return "the SMTP server rejected the login, so the SMTP username or password is wrong (" + raw + ")"
	case strings.Contains(lower, "connection refused") || strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "i/o timeout") || strings.Contains(lower, "network is unreachable"):
		return "couldn't reach the SMTP server (" + raw + ")"
	case strings.Contains(lower, "certificate") || strings.Contains(lower, "tls"):
		return "the SMTP server's TLS setup didn't match (" + raw + ")"
	case strings.HasPrefix(raw, "550") || strings.HasPrefix(raw, "553") || strings.HasPrefix(raw, "554"):
		return "the SMTP server refused the message (" + raw + ")"
	}
	return raw
}
