package inboxcheck

import (
	"strings"
	"testing"
)

// failing returns the IDs of the checks that didn't pass.
func failing(checks []Check) map[string]Check {
	out := map[string]Check{}
	for _, c := range checks {
		if !c.Pass {
			out[c.ID] = c
		}
	}
	return out
}

// good is an email that passes everything.
func good() Input {
	return Input{
		Subject:       "Our October update",
		FromEmail:     "Acme <news@acme.com>",
		WorkspaceFrom: "Acme <hello@acme.com>",
		HTML: `<html><head><style>p{color:red}</style></head><body>
			<p>` + strings.Repeat("Real words about the product. ", 12) + `</p>
			<img src="https://acme.com/hero.png" width="600">
			<a href="https://acme.com/post">Read more</a>
			<a href="mailto:help@acme.com">Email us</a>
			<a href="https://listnun.app/subscription/a/b">Unsubscribe</a>
			<img src="https://listnun.app/campaign/a/b/px.png" width="1" height="1">
		</body></html>`,
		HasPlainText: true,
		Audience:     1000, Inactive: 100,
		Sent: 5000, HardBounces: 10, Complaints: 2,
	}
}

func TestRun_GoodEmailPasses(t *testing.T) {
	if f := failing(Run(good())); len(f) != 0 {
		t.Fatalf("failing checks = %v, want none", f)
	}
}

func TestRun_FlagsEachProblem(t *testing.T) {
	cases := []struct {
		name   string
		edit   func(*Input)
		want   string
		params map[string]string
	}{
		{"no text version", func(in *Input) { in.HasPlainText = false }, "plainText", nil},
		{"clipped by Gmail", func(in *Input) { in.HTML += strings.Repeat("<p>x</p>", 15000) }, "size", nil},
		{"mostly images", func(in *Input) {
			in.HTML = `<body><img src="a.png"><img src="b.png"><p>Sale</p></body>`
		}, "images", map[string]string{"n": "2"}},
		{"link to nowhere", func(in *Input) { in.HTML += `<a href="#">Shop</a><a href="www.acme.com">Site</a>` }, "links", map[string]string{"n": "2"}},
		{"link shortener", func(in *Input) { in.HTML += `<a href="https://bit.ly/x">Deal</a><a href="https://bit.ly/y">Deal</a>` },
			"shorteners", map[string]string{"hosts": "bit.ly"}},
		{"empty subject", func(in *Input) { in.Subject = "  " }, "subjectLength", nil},
		{"long subject", func(in *Input) { in.Subject = strings.Repeat("word ", 14) }, "subjectLength", nil},
		{"shouting subject", func(in *Input) { in.Subject = "HUGE SALE THIS WEEK" }, "subjectCaps", nil},
		{"repeated punctuation", func(in *Input) { in.Subject = "Last chance!!" }, "subjectPunct", nil},
		{"spammy phrase", func(in *Input) { in.Subject = "Act now: 100% FREE gift" }, "subjectSpam",
			map[string]string{"phrases": "100% free, act now, free gift"}},
		{"free-mail From", func(in *Input) { in.FromEmail = "Acme <acme.news@gmail.com>" }, "fromFreeMail", map[string]string{"domain": "gmail.com"}},
		{"From off the sending domain", func(in *Input) { in.FromEmail = "news@other.com" }, "fromDomain",
			map[string]string{"domain": "other.com", "expected": "acme.com"}},
		{"mostly inactive", func(in *Input) { in.Inactive = 620 }, "inactive", map[string]string{"pct": "62%", "n": "620", "days": "90"}},
		{"complaints", func(in *Input) { in.Complaints = 20 }, "complaints", map[string]string{"pct": "0.4%", "days": "30"}},
		{"bounces", func(in *Input) { in.HardBounces = 150 }, "bounces", map[string]string{"pct": "3%", "days": "30"}},
	}
	for _, tc := range cases {
		in := good()
		tc.edit(&in)
		f := failing(Run(in))
		got, ok := f[tc.want]
		if !ok || len(f) != 1 {
			t.Errorf("%s: failing = %v, want only %s", tc.name, f, tc.want)
			continue
		}
		for k, v := range tc.params {
			if got.Params[k] != v {
				t.Errorf("%s: params[%s] = %q, want %q", tc.name, k, got.Params[k], v)
			}
		}
	}
}

func TestRun_SkipsWhatItCantJudge(t *testing.T) {
	in := good()
	in.HTML = "" // a plain-text campaign
	in.WorkspaceFrom = ""
	in.Audience = 0
	in.Sent = 40 // too few sends for a rate to mean anything
	in.Complaints = 5

	seen := map[string]bool{}
	for _, c := range Run(in) {
		seen[c.ID] = true
	}
	for _, id := range []string{"size", "images", "links", "shorteners", "fromDomain", "inactive", "complaints", "bounces"} {
		if seen[id] {
			t.Errorf("ran %s, which has nothing to judge here", id)
		}
	}
}

// A subdomain of the workspace's sending domain is covered by it, and
// short all-caps words like "FAQ" aren't shouting.
func TestRun_AllowsCommonCases(t *testing.T) {
	in := good()
	in.FromEmail = "news@mail.acme.com"
	in.Subject = "New FAQ and API docs"
	if f := failing(Run(in)); len(f) != 0 {
		t.Fatalf("failing = %v, want none", f)
	}
}
