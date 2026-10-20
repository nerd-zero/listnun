package captcha

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/altcha-org/altcha-lib-go"
	"github.com/nerd-zero/listnun/internal/tmptokens"
)

const (
	hCaptchaURL  = "https://hcaptcha.com/siteverify"
	turnstileURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
)

type hCaptchaResp struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error_codes"`
}

// turnstileResp is Cloudflare Turnstile's siteverify response.
type turnstileResp struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

const (
	ProviderNone      = ""
	ProviderHCaptcha  = "hcaptcha"
	ProviderAltcha    = "altcha"
	ProviderTurnstile = "turnstile"
)

// Captcha is a captcha client supporting multiple providers.
type Captcha struct {
	provider  string
	hCaptcha  hCaptchaOpt
	altcha    altchaOpt
	turnstile turnstileOpt
	onLogin   bool
	client    *http.Client
}

type Opt struct {
	HCaptcha struct {
		Enabled bool   `json:"enabled"`
		Key     string `json:"key"`
		Secret  string `json:"secret"`
	} `json:"hcaptcha"`
	Altcha struct {
		Enabled    bool `json:"enabled"`
		Complexity int  `json:"complexity"`
	} `json:"altcha"`
	Turnstile struct {
		Enabled bool   `json:"enabled"`
		Key     string `json:"key"`
		Secret  string `json:"secret"`
	} `json:"turnstile"`
	// Login also asks for the captcha on the admin sign-in page.
	Login bool `json:"login"`
}

type hCaptchaOpt struct {
	Secret string
}

type turnstileOpt struct {
	Secret string
	// URL is the siteverify endpoint; tests point it at a fake server.
	URL string
}

type altchaOpt struct {
	Complexity int
	HMACKey    string
}

// New returns a new instance of the CAPTCHA client.
func New(o Opt) *Captcha {
	timeout := time.Second * 5

	c := &Captcha{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConnsPerHost:   10,
				MaxConnsPerHost:       100,
				ResponseHeaderTimeout: timeout,
				IdleConnTimeout:       timeout,
			},
		},
	}

	// Determine which provider is enabled
	if o.Altcha.Enabled {
		c.provider = ProviderAltcha

		// Generate an random HMAC key for Altcha.
		b := make([]byte, 24) // 24 bytes will give 32 characters when base64 encoded
		_, err := rand.Read(b)
		if err != nil {
			panic(fmt.Sprintf("error generating Altcha HMAC key: %v", err))
		}
		hmacKey := base64.URLEncoding.EncodeToString(b)[:32]

		c.altcha = altchaOpt{
			Complexity: o.Altcha.Complexity,
			HMACKey:    hmacKey,
		}
	} else if o.Turnstile.Enabled {
		c.provider = ProviderTurnstile
		c.turnstile = turnstileOpt{Secret: o.Turnstile.Secret, URL: turnstileURL}
	} else if o.HCaptcha.Enabled {
		c.provider = ProviderHCaptcha
		c.hCaptcha = hCaptchaOpt{
			Secret: o.HCaptcha.Secret,
		}
	}
	c.onLogin = o.Login && c.provider != ProviderNone

	return c
}

// IsEnabled returns true if any captcha provider is enabled.
func (c *Captcha) IsEnabled() bool {
	return c.provider != ProviderNone
}

// OnLogin reports whether the admin sign-in page asks for the captcha too.
func (c *Captcha) OnLogin() bool {
	return c.onLogin
}

// FormField is the name of the form field the active provider's widget
// puts its response in.
func (c *Captcha) FormField() string {
	switch c.provider {
	case ProviderAltcha:
		return "altcha"
	case ProviderHCaptcha:
		return "h-captcha-response"
	case ProviderTurnstile:
		return "cf-turnstile-response"
	default:
		return ""
	}
}

// GetProvider returns the active captcha provider.
func (c *Captcha) GetProvider() string {
	return c.provider
}

// GenerateChallenge generates a challenge for the active provider.
// For hCaptcha, this returns empty string as challenges are generated client-side.
// For Altcha, this returns a JSON challenge.
func (c *Captcha) GenerateChallenge() (string, error) {
	switch c.provider {
	case ProviderAltcha:
		exp := time.Now().Add(5 * time.Minute)
		challenge, err := altcha.CreateChallenge(altcha.ChallengeOptions{
			Algorithm:  altcha.SHA256,
			MaxNumber:  int64(c.altcha.Complexity),
			SaltLength: 12,
			HMACKey:    c.altcha.HMACKey,
			Expires:    &exp,
		})
		if err != nil {
			return "", fmt.Errorf("failed to create Altcha challenge: %w", err)
		}

		challengeJSON, err := json.Marshal(challenge)
		if err != nil {
			return "", fmt.Errorf("failed to marshal Altcha challenge: %w", err)
		}

		return string(challengeJSON), nil
	case ProviderHCaptcha, ProviderTurnstile:
		// hCaptcha and Turnstile generate challenges client-side.
		return "", nil
	default:
		return "", fmt.Errorf("no captcha provider enabled")
	}
}

// Verify verifies a CAPTCHA response.
func (c *Captcha) Verify(token string) (error, bool) {
	switch c.provider {
	case ProviderAltcha:
		return c.verifyAltcha(token)
	case ProviderHCaptcha:
		return c.verifyHCaptcha(token)
	case ProviderTurnstile:
		return c.verifyTurnstile(token)
	default:
		return fmt.Errorf("no captcha provider enabled"), false
	}
}

// verifyHCaptcha verifies an hCaptcha response.
func (c *Captcha) verifyHCaptcha(token string) (error, bool) {
	resp, err := c.client.PostForm(hCaptchaURL, url.Values{
		"secret":   {c.hCaptcha.Secret},
		"response": {token},
	})
	if err != nil {
		return err, false
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err, false
	}

	var r hCaptchaResp
	if err := json.Unmarshal(body, &r); err != nil {
		// Fail closed: an unreadable answer is not a pass.
		return err, false
	}

	if !r.Success {
		return fmt.Errorf("hCaptcha failed: %s", strings.Join(r.ErrorCodes, ",")), false
	}

	return nil, true
}

// verifyTurnstile verifies a Cloudflare Turnstile response. Cloudflare
// rejects a token that was already redeemed, so reuse needs no local check.
func (c *Captcha) verifyTurnstile(token string) (error, bool) {
	resp, err := c.client.PostForm(c.turnstile.URL, url.Values{
		"secret":   {c.turnstile.Secret},
		"response": {token},
	})
	if err != nil {
		return err, false
	}
	defer resp.Body.Close()

	var r turnstileResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return err, false
	}
	if !r.Success {
		return fmt.Errorf("turnstile failed: %s", strings.Join(r.ErrorCodes, ",")), false
	}

	return nil, true
}

// verifyAltcha verifies an Altcha response.
func (c *Captcha) verifyAltcha(payload string) (error, bool) {
	valid, err := altcha.VerifySolution(payload, c.altcha.HMACKey, true)
	if err != nil {
		return fmt.Errorf("failed to verify captcha solution: %w", err), false
	}

	if !valid {
		return fmt.Errorf("captcha verification failed"), false
	}

	// Disallow token reuse.
	if _, err := tmptokens.Check(payload); err == nil {
		return fmt.Errorf("captcha token already used"), false
	}
	tmptokens.Set(payload, 5*time.Minute, nil)

	return nil, true
}
