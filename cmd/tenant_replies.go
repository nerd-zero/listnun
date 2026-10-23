package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/nerd-zero/listnun/internal/replies"
	"github.com/nerd-zero/listnun/models"
)

// tenantRepliesTTL is how long a tenant's reply tracking setup is cached
// for the send path. Short, so a setup pushed through the Operator API
// takes effect on other replicas within a minute without a restart; the
// replica that received the push drops its entry at once (invalidate).
const tenantRepliesTTL = time.Minute

// tenantReplies resolves each tenant's reply tracking setup
// (models.TenantReplies, its "replies" settings row): cached for the
// campaign send path, which asks once per message, and read fresh by the
// inbound webhook, which checks the credentials.
type tenantReplies struct {
	q   *models.Queries
	log *log.Logger

	mu    sync.Mutex
	cache map[int]tenantRepliesEntry
}

type tenantRepliesEntry struct {
	addr    *replies.Addresser // nil: reply tracking off for the tenant
	fetched time.Time
}

func newTenantReplies(q *models.Queries, lo *log.Logger) *tenantReplies {
	return &tenantReplies{q: q, log: lo, cache: map[int]tenantRepliesEntry{}}
}

// Config returns tenantID's reply tracking setup and its Addresser, read
// fresh from the database. ok is false when the tenant has none.
func (t *tenantReplies) Config(tenantID int) (cfg models.TenantReplies, addr *replies.Addresser, ok bool, err error) {
	var raw json.RawMessage
	if err := t.q.GetTenantReplies.Get(&raw, tenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return cfg, nil, false, nil
		}
		return cfg, nil, false, err
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, nil, false, err
	}
	addr, err = replies.NewTenantAddresser(cfg)
	if err != nil {
		return cfg, nil, false, err
	}
	return cfg, addr, true, nil
}

// Addresser returns tenantID's cached Addresser, or nil when reply
// tracking is off (or its setup can't be read -- logged, and sending goes
// on with the campaign's own Reply-To rather than failing).
func (t *tenantReplies) Addresser(tenantID int) *replies.Addresser {
	t.mu.Lock()
	e, ok := t.cache[tenantID]
	t.mu.Unlock()
	if ok && time.Since(e.fetched) < tenantRepliesTTL {
		return e.addr
	}

	_, addr, _, err := t.Config(tenantID)
	if err != nil {
		t.log.Printf("replies: error reading tenant %d's reply setup, tracking off for it: %v", tenantID, err)
		addr = nil
	}
	t.mu.Lock()
	t.cache[tenantID] = tenantRepliesEntry{addr: addr, fetched: time.Now()}
	t.mu.Unlock()
	return addr
}

// ReplyAddress is the manager's Config.ReplyAddress: the reply address for
// one message, or "" when the tenant has no reply tracking setup.
func (t *tenantReplies) ReplyAddress(tenantID, campaignID, subscriberID int) string {
	addr := t.Addresser(tenantID)
	if addr == nil {
		return ""
	}
	return addr.Address(replies.Ref{TenantID: tenantID, CampaignID: campaignID, SubscriberID: subscriberID})
}

// invalidate drops tenantID's cached setup after the Operator API changes it.
func (t *tenantReplies) invalidate(tenantID int) {
	t.mu.Lock()
	delete(t.cache, tenantID)
	t.mu.Unlock()
}
