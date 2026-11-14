package manager

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/nerd-zero/listnun/models"
)

// linkStore is a fake Store that hands out a distinct link UUID per
// (tenant, URL), the way each tenant's links rows are its own.
type linkStore struct {
	Store   // unimplemented methods panic, which the test would catch
	created []string
}

func (s *linkStore) CreateLink(_ context.Context, tenantID int, url string) (string, error) {
	s.created = append(s.created, fmt.Sprintf("%d:%s", tenantID, url))
	return fmt.Sprintf("uuid-%d-%d", tenantID, len(s.created)), nil
}

func (s *linkStore) GetTenantRootURL(tenantID int) (string, error) {
	return fmt.Sprintf("https://t%d.example.com", tenantID), nil
}

func newLinkTestManager(st Store) *Manager {
	return &Manager{
		cfg:        Config{Tracking: func(int) Tracking { return Tracking{} }},
		store:      st,
		links:      make(map[linkKey]string),
		tenantURLs: make(map[int]tenantURLCfg),
		log:        log.New(io.Discard, "", 0),
	}
}

// The link cache used to be keyed by URL alone, so a second tenant linking
// to the same URL got the first tenant's link UUID, which its own click
// handler can't see.
func TestTrackLinkCachesPerTenant(t *testing.T) {
	st := &linkStore{}
	m := newLinkTestManager(st)

	a1 := m.trackLink(1, "https://listnun.app", "camp", "sub")
	a2 := m.trackLink(1, "https://listnun.app", "camp", "sub")
	b := m.trackLink(2, "https://listnun.app", "camp", "sub")

	if a1 != a2 {
		t.Errorf("same tenant and URL gave two links: %q, %q", a1, a2)
	}
	if len(st.created) != 2 {
		t.Fatalf("CreateLink calls = %v, want one per tenant", st.created)
	}
	if !strings.HasPrefix(b, "https://t2.example.com/link/uuid-2-") {
		t.Errorf("tenant 2 got %q, want its own link on its own root URL", b)
	}
}

// Tracking settings are per tenant. They used to come from the boot-time
// config (tenant 1's), so another tenant turning on per-subscriber
// tracking still sent placeholder subscribers and counted every click.
func TestTrackingSettingsPerTenant(t *testing.T) {
	m := newLinkTestManager(&linkStore{})
	m.cfg.Tracking = func(tenantID int) Tracking {
		switch tenantID {
		case 2:
			return Tracking{Individual: true}
		case 3:
			return Tracking{Disabled: true}
		}
		return Tracking{}
	}

	msg := func(tenantID int) *CampaignMessage {
		c := &models.Campaign{TenantID: tenantID}
		c.UUID = "camp-uuid"
		s := models.Subscriber{}
		s.UUID = "sub-uuid"
		return &CampaignMessage{Campaign: c, Subscriber: s}
	}

	cases := []struct {
		tenantID     int
		wantLink     string
		wantPixelHas string
		wantNoPixel  bool
	}{
		{1, "https://t1.example.com/link/uuid-1-1/camp-uuid/" + dummyUUID, "/camp-uuid/" + dummyUUID + "/px.png", false},
		{2, "https://t2.example.com/link/uuid-2-2/camp-uuid/sub-uuid", "/camp-uuid/sub-uuid/px.png", false},
		{3, "https://listnun.app", "", true},
	}
	for _, tc := range cases {
		c := msg(tc.tenantID)
		f := m.TemplateFuncs(c.Campaign)

		link := f["TrackLink"].(func(string, *CampaignMessage) string)("https://listnun.app", c)
		if link != tc.wantLink {
			t.Errorf("tenant %d: TrackLink = %q, want %q", tc.tenantID, link, tc.wantLink)
		}

		px := string(f["TrackView"].(func(*CampaignMessage) template.HTML)(c))
		if tc.wantNoPixel {
			if px != "" {
				t.Errorf("tenant %d: TrackView = %q, want no pixel", tc.tenantID, px)
			}
		} else if !strings.Contains(px, tc.wantPixelHas) {
			t.Errorf("tenant %d: TrackView = %q, want it to contain %q", tc.tenantID, px, tc.wantPixelHas)
		}
	}
}

// Campaign status notifications went to tenant 1's notify addresses for
// every workspace's campaigns; they now carry the campaign's workspace.
func TestSendNotifCarriesTheCampaignsTenant(t *testing.T) {
	var gotTenant int
	m := &Manager{fnNotify: func(tenantID int, subject string, data any) error {
		gotTenant = tenantID
		return nil
	}}
	c := &models.Campaign{TenantID: 7, Name: "Launch"}
	if err := m.sendNotif(c, "finished", ""); err != nil {
		t.Fatal(err)
	}
	if gotTenant != 7 {
		t.Errorf("notification for tenant %d, want the campaign's 7", gotTenant)
	}

	// With no notifier configured, nothing is sent and nothing fails.
	if err := (&Manager{}).sendNotif(c, "finished", ""); err != nil {
		t.Errorf("sendNotif with no notifier: %v", err)
	}
}
