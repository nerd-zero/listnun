// Package inboxcheck reviews a campaign before it's sent for what commonly
// sends email to spam or the promotions tab: its content, subject line,
// sender and audience. It only reads what it's given, so every rule is
// testable without a database; internal/core gathers the inputs.
package inboxcheck

import (
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

// Check groups, in the order the editor shows them.
const (
	GroupContent  = "content"
	GroupSubject  = "subject"
	GroupSender   = "sender"
	GroupAudience = "audience"
)

// Check is one inbox check's outcome. The editor words it from ID and
// Params (the i18n keys campaigns.inbox.<id> and campaigns.inbox.<id>Warn),
// so the API stays language-neutral.
type Check struct {
	ID     string            `json:"id"`
	Group  string            `json:"group"`
	Pass   bool              `json:"pass"`
	Params map[string]string `json:"params,omitempty"` // Open-ended: each check's own placeholders.
}

// Input is everything the checks read.
type Input struct {
	Subject string
	// FromEmail is the campaign's From header, e.g. "Acme <news@acme.com>".
	FromEmail string
	// WorkspaceFrom is the workspace's default From, the address its sending
	// is set up for (listnun points it at the verified sender).
	WorkspaceFrom string
	// HTML is the email as a subscriber gets it, template included; empty
	// for a plain-text campaign.
	HTML string
	// HasPlainText is whether the email carries a text version: a
	// plain-text campaign, or an HTML one with an alternate body.
	HasPlainText bool

	// Audience is how many subscribers the campaign goes to, and Inactive
	// how many of them haven't opened or clicked in InactiveDays.
	Audience, Inactive int
	// Sent, HardBounces and Complaints are the workspace's campaign
	// messages, hard bounces and spam complaints over the last RecentDays.
	Sent, HardBounces, Complaints int
}

// Thresholds. Each is where mailbox providers start treating mail worse,
// not a style preference.
const (
	// GmailClipBytes is where Gmail cuts an email off behind "[Message
	// clipped]", hiding the rest, including the unsubscribe link.
	GmailClipBytes = 102 * 1024

	// minTextPerImage is the visible text, in characters, below which an
	// email reads as mostly images: filters can't judge it and many readers
	// see nothing until they allow images.
	minTextPerImage = 200

	// maxSubjectRunes is roughly what phones show of a subject line.
	maxSubjectRunes = 60

	// InactiveDays is the window for "hasn't engaged" (the same 90 days the
	// inactive cleanup suggests). maxInactiveShare is the share of
	// unengaged subscribers above which Gmail and Yahoo start reading the
	// sender as unwanted.
	InactiveDays     = 90
	maxInactiveShare = 0.5

	// RecentDays is the window for the workspace's bounce and complaint
	// rates. Google asks bulk senders to stay under 0.1% complaints (0.3% is
	// the hard limit), and more than 2% hard bounces is a stale list.
	RecentDays       = 30
	maxComplaintRate = 0.001
	maxBounceRate    = 0.02
	// minRateSample is how many sends a rate needs before it means anything.
	minRateSample = 100
)

// freeMailDomains publish strict DMARC policies, so mail "from" them sent
// through any other server fails authentication and is rejected or
// spam-foldered.
var freeMailDomains = map[string]bool{
	"gmail.com": true, "googlemail.com": true, "yahoo.com": true, "ymail.com": true,
	"outlook.com": true, "hotmail.com": true, "live.com": true, "msn.com": true,
	"aol.com": true, "icloud.com": true, "me.com": true, "mac.com": true,
	"gmx.com": true, "proton.me": true, "protonmail.com": true, "zoho.com": true,
	"yandex.com": true, "mail.com": true,
}

// shortenerHosts hide where a link goes, which is what spammers use them
// for, so filters penalise them.
var shortenerHosts = map[string]bool{
	"bit.ly": true, "tinyurl.com": true, "t.co": true, "goo.gl": true, "ow.ly": true,
	"is.gd": true, "buff.ly": true, "rebrand.ly": true, "cutt.ly": true, "shorturl.at": true,
	"tiny.cc": true, "rb.gy": true, "s.id": true,
}

// spamPhrases are phrases content filters score heavily, matched
// case-insensitively in the subject line.
var spamPhrases = []string{
	"100% free", "act now", "buy now", "cash bonus", "click here", "double your",
	"earn money", "free gift", "guaranteed", "limited time", "no cost", "order now",
	"risk-free", "risk free", "urgent", "winner", "you have been selected", "you've won",
	"call now", "apply now", "credit card", "make money", "lowest price", "miracle",
}

// Run returns every check's outcome, grouped in display order.
func Run(in Input) []Check {
	var out []Check
	out = append(out, contentChecks(in)...)
	out = append(out, subjectChecks(in.Subject)...)
	out = append(out, senderChecks(in.FromEmail, in.WorkspaceFrom)...)
	out = append(out, audienceChecks(in)...)
	return out
}

func check(id, group string, pass bool, params map[string]string) Check {
	return Check{ID: id, Group: group, Pass: pass, Params: params}
}

func contentChecks(in Input) []Check {
	out := []Check{check("plainText", GroupContent, in.HasPlainText, nil)}
	if in.HTML == "" {
		return out
	}

	size := len(in.HTML)
	out = append(out, check("size", GroupContent, size <= GmailClipBytes,
		map[string]string{"kb": fmt.Sprint((size + 1023) / 1024)}))

	doc := scan(in.HTML)
	out = append(out, check("images", GroupContent, doc.images == 0 || doc.textRunes >= minTextPerImage*doc.images,
		map[string]string{"n": fmt.Sprint(doc.images)}))
	out = append(out, check("links", GroupContent, doc.brokenLinks == 0,
		map[string]string{"n": fmt.Sprint(doc.brokenLinks)}))
	out = append(out, check("shorteners", GroupContent, len(doc.shorteners) == 0,
		map[string]string{"hosts": strings.Join(doc.shorteners, ", ")}))
	return out
}

// htmlStats is what the content checks need from an email's HTML.
type htmlStats struct {
	textRunes   int
	images      int
	brokenLinks int
	shorteners  []string
}

// scan walks the email's HTML once. Template tags like {{ UnsubscribeURL }}
// are already rendered by the time it gets here.
func scan(src string) htmlStats {
	var (
		st   htmlStats
		seen = map[string]bool{}
		skip int // depth inside <style>/<script>/<head>, whose text isn't shown
	)
	z := html.NewTokenizer(strings.NewReader(src))
	for {
		switch z.Next() {
		case html.ErrorToken:
			return st
		case html.TextToken:
			if skip == 0 {
				for _, r := range string(z.Text()) {
					if !unicode.IsSpace(r) {
						st.textRunes++
					}
				}
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			tok := z.Token()
			switch tok.Data {
			case "style", "script", "head", "title":
				if tok.Type == html.StartTagToken {
					skip++
				}
			case "img":
				if !isTrackingPixel(tok) {
					st.images++
				}
			case "a":
				href, ok := attr(tok, "href")
				if !ok {
					continue
				}
				href = strings.TrimSpace(href)
				if isBrokenHref(href) {
					st.brokenLinks++
					continue
				}
				if u, err := url.Parse(href); err == nil {
					host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
					if shortenerHosts[host] && !seen[host] {
						seen[host] = true
						st.shorteners = append(st.shorteners, host)
					}
				}
			}
		case html.EndTagToken:
			switch tok := z.Token(); tok.Data {
			case "style", "script", "head", "title":
				if skip > 0 {
					skip--
				}
			}
		}
	}
}

func attr(tok html.Token, name string) (string, bool) {
	for _, a := range tok.Attr {
		if a.Key == name {
			return a.Val, true
		}
	}
	return "", false
}

// isTrackingPixel is a 1x1 image (the open-tracking pixel and its kind),
// which isn't content.
func isTrackingPixel(tok html.Token) bool {
	w, _ := attr(tok, "width")
	h, _ := attr(tok, "height")
	return strings.TrimSuffix(w, "px") == "1" && strings.TrimSuffix(h, "px") == "1"
}

// isBrokenHref is a link that goes nowhere: empty, a bare "#", a
// placeholder left in, or a web address without its scheme.
func isBrokenHref(href string) bool {
	switch strings.ToLower(href) {
	case "", "#", "http://", "https://", "url", "link":
		return true
	}
	lower := strings.ToLower(href)
	for _, p := range []string{"http:", "https:", "mailto:", "tel:", "sms:", "#", "/"} {
		if strings.HasPrefix(lower, p) {
			return false
		}
	}
	// "www.example.com" with no scheme opens relative to the email client.
	return true
}

func subjectChecks(subject string) []Check {
	s := strings.TrimSpace(subject)
	out := []Check{check("subjectLength", GroupSubject, s != "" && len([]rune(s)) <= maxSubjectRunes,
		map[string]string{"n": fmt.Sprint(len([]rune(s))), "max": fmt.Sprint(maxSubjectRunes)})}

	var letters, upper int
	for _, r := range s {
		if unicode.IsLetter(r) {
			letters++
			if unicode.IsUpper(r) {
				upper++
			}
		}
	}
	out = append(out, check("subjectCaps", GroupSubject, letters < 8 || upper*2 < letters, nil))
	out = append(out, check("subjectPunct", GroupSubject, !hasRepeats(s, "!?$€£"), nil))

	var found []string
	lower := strings.ToLower(s)
	for _, p := range spamPhrases {
		if strings.Contains(lower, p) {
			found = append(found, p)
		}
	}
	out = append(out, check("subjectSpam", GroupSubject, len(found) == 0,
		map[string]string{"phrases": strings.Join(found, ", ")}))
	return out
}

// hasRepeats is whether s has two of the same character from chars in a
// row, like "!!" or "$$".
func hasRepeats(s, chars string) bool {
	var prev rune
	for _, r := range s {
		if r == prev && strings.ContainsRune(chars, r) {
			return true
		}
		prev = r
	}
	return false
}

func senderChecks(from, workspaceFrom string) []Check {
	domain := emailDomain(from)
	out := []Check{check("fromFreeMail", GroupSender, domain == "" || !freeMailDomains[domain],
		map[string]string{"domain": domain})}

	// Only meaningful once the workspace's sending is set up for an address.
	expected := emailDomain(workspaceFrom)
	if expected != "" && domain != "" && !freeMailDomains[domain] {
		out = append(out, check("fromDomain", GroupSender, domain == expected || strings.HasSuffix(domain, "."+expected),
			map[string]string{"domain": domain, "expected": expected}))
	}
	return out
}

// emailDomain is the lowercased domain of an address or "Name <address>".
func emailDomain(from string) string {
	addr := strings.TrimSpace(from)
	if a, err := mail.ParseAddress(addr); err == nil {
		addr = a.Address
	}
	i := strings.LastIndex(addr, "@")
	if i < 0 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(addr[i+1:]))
}

func audienceChecks(in Input) []Check {
	var out []Check
	if in.Audience > 0 {
		share := float64(in.Inactive) / float64(in.Audience)
		out = append(out, check("inactive", GroupAudience, share <= maxInactiveShare,
			map[string]string{"pct": percent(share), "n": fmt.Sprint(in.Inactive), "days": fmt.Sprint(InactiveDays)}))
	}
	if r := RecentRates(in.Sent, in.HardBounces, in.Complaints); r.Judged {
		out = append(out, check("complaints", GroupAudience, !r.ComplaintsHigh,
			map[string]string{"pct": percent(r.ComplaintRate), "days": fmt.Sprint(RecentDays)}))
		out = append(out, check("bounces", GroupAudience, !r.BouncesHigh,
			map[string]string{"pct": percent(r.BounceRate), "days": fmt.Sprint(RecentDays)}))
	}
	return out
}

// Rates is a workspace's complaint and hard-bounce rates over RecentDays,
// judged against what Gmail and Yahoo tolerate.
type Rates struct {
	ComplaintRate float64 `json:"complaint_rate"`
	BounceRate    float64 `json:"bounce_rate"`
	// Judged is false below minRateSample sends, when the rates say
	// nothing yet and the High flags stay false.
	Judged         bool `json:"judged"`
	ComplaintsHigh bool `json:"complaints_high"`
	BouncesHigh    bool `json:"bounces_high"`
}

// RecentRates rates sent campaign messages' hard bounces and complaints.
func RecentRates(sent, hardBounces, complaints int) Rates {
	if sent <= 0 {
		return Rates{}
	}
	r := Rates{
		ComplaintRate: float64(complaints) / float64(sent),
		BounceRate:    float64(hardBounces) / float64(sent),
		Judged:        sent >= minRateSample,
	}
	if r.Judged {
		r.ComplaintsHigh = r.ComplaintRate > maxComplaintRate
		r.BouncesHigh = r.BounceRate > maxBounceRate
	}
	return r
}

// percent formats a share for a sentence: "0.4%", "62%".
func percent(share float64) string {
	p := share * 100
	if p > 0 && p < 10 {
		return strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintf("%.1f", p), "0"), ".") + "%"
	}
	return fmt.Sprintf("%.0f%%", p)
}
