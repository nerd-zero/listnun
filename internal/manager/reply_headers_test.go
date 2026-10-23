package manager

import (
	"fmt"
	"testing"

	"github.com/nerd-zero/listnun/models"
)

func replyTestMessage(trackReplies bool) CampaignMessage {
	c := &models.Campaign{UUID: "camp-uuid", TenantID: 7, Tracking: models.CampaignTracking{Replies: trackReplies}}
	c.ID = 42
	s := models.Subscriber{UUID: "sub-uuid"}
	s.ID = 1001
	return CampaignMessage{
		Campaign:   c,
		Subscriber: s,
		headers:    models.Headers{{"Reply-To": "owner@fuser.tech"}, {"X-Custom": "kept"}},
	}
}

func replyTestManager(withAddress bool) *Manager {
	m := &Manager{}
	if withAddress {
		m.cfg.ReplyAddress = func(tenantID, campaignID, subscriberID int) string {
			return fmt.Sprintf("hash+t%d-c%d-s%d@inbound.example", tenantID, campaignID, subscriberID)
		}
	}
	return m
}

func TestCampaignHeadersReplyTrackingSwapsReplyTo(t *testing.T) {
	h := replyTestManager(true).campaignHeaders(replyTestMessage(true))

	if got := h.Values("Reply-To"); len(got) != 1 || got[0] != "hash+t7-c42-s1001@inbound.example" {
		t.Errorf("Reply-To = %q, want only the tenant/campaign/subscriber reply address", got)
	}
	if h.Get("X-Custom") != "kept" || h.Get(models.EmailHeaderCampaignUUID) != "camp-uuid" {
		t.Errorf("other headers changed: %v", h)
	}
}

func TestCampaignHeadersKeepReplyToWhenTrackingOff(t *testing.T) {
	for name, m := range map[string]struct {
		mgr   *Manager
		track bool
	}{
		"campaign switch off":   {replyTestManager(true), false},
		"server not configured": {replyTestManager(false), true},
	} {
		h := m.mgr.campaignHeaders(replyTestMessage(m.track))
		if h.Get("Reply-To") != "owner@fuser.tech" {
			t.Errorf("%s: Reply-To = %q, want the campaign's own", name, h.Get("Reply-To"))
		}
	}
}
