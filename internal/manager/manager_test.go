package manager

import (
	"context"
	"fmt"
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
