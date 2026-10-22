package manager

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"
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
