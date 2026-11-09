package models

import "testing"

func TestCampaignHasUnsubscribeLink(t *testing.T) {
	cases := []struct {
		name string
		c    Campaign
		want bool
	}{
		{"in the template", Campaign{ContentType: CampaignContentTypeRichtext,
			TemplateBody: `{{ template "content" . }}<a href="{{ UnsubscribeURL }}">Unsubscribe</a>`, Body: "<p>Hi</p>"}, true},
		{"in the body", Campaign{ContentType: CampaignContentTypeHTML,
			TemplateBody: `{{ template "content" . }}`, Body: `<a href="{{UnsubscribeURL}}">Leave</a>`}, true},
		{"preferences link counts", Campaign{ContentType: CampaignContentTypeMarkdown,
			Body: "[Manage your emails]({{ ManageURL }})"}, true},
		{"trimmed tag", Campaign{ContentType: CampaignContentTypePlain,
			Body: "Unsubscribe: {{- UnsubscribeURL -}}"}, true},
		{"no link anywhere", Campaign{ContentType: CampaignContentTypeRichtext,
			TemplateBody: `<html>{{ template "content" . }}</html>`, Body: "<p>Buy now</p>"}, false},
		{"only the words", Campaign{ContentType: CampaignContentTypeRichtext,
			Body: "<p>Reply STOP to unsubscribe</p>"}, false},
		// A visual campaign renders without its template.
		{"visual ignores the template", Campaign{ContentType: CampaignContentTypeVisual,
			TemplateBody: `<a href="{{ UnsubscribeURL }}">x</a>`, Body: "<table>Hi</table>"}, false},
		{"visual footer block", Campaign{ContentType: CampaignContentTypeVisual,
			Body: `<td><a href="{{ UnsubscribeURL }}" style="color: #888;">Unsubscribe</a></td>`}, true},
	}
	for _, tc := range cases {
		if got := tc.c.HasUnsubscribeLink(); got != tc.want {
			t.Errorf("%s: HasUnsubscribeLink() = %v, want %v", tc.name, got, tc.want)
		}
	}
}
