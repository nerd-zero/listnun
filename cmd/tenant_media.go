package main

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/listmonk/internal/core"
	"github.com/knadh/listmonk/internal/media"
	"github.com/knadh/listmonk/models"
)

// tenantMediaStore pairs a resolved media.Store with the settings it was
// built from, so callers can also read upload.provider/upload.extensions
// (queries/media.sql's provider column, UploadMedia's extension check)
// without a second settings lookup.
type tenantMediaStore struct {
	store    media.Store
	settings models.Settings
}

// tenantMedia lazily builds and caches media.Store instances per tenant
// from that tenant's own settings (settings are per-tenant since phase 5),
// the same pattern as tenantMessengers (see cmd/tenant_messenger.go) for
// SMTP. Constructed once and held by *App and cmd/manager_store.go's
// *store; always resolves against the DB rather than a settings snapshot.
//
// A cache entry is evicted (see Invalidate) whenever the owning tenant
// saves upload/media settings, so it's rebuilt from the DB on next use
// instead of requiring the full process restart the rest of
// cmd/settings.go's handleSettingsRestart still uses for settings groups
// outside this, tenantMessengers', and internal/auth's OIDC cache.
type tenantMedia struct {
	core *core.Core

	mu    sync.Mutex
	cache map[int]tenantMediaStore
}

func newTenantMedia(co *core.Core) *tenantMedia {
	return &tenantMedia{
		core:  co,
		cache: make(map[int]tenantMediaStore),
	}
}

// Invalidate evicts the given tenant's cached media.Store, forcing the
// next Get call to rebuild it from that tenant's current settings instead
// of requiring a full process restart (cmd/settings.go's
// handleSettingsRestart).
func (t *tenantMedia) Invalidate(tenantID int) {
	t.mu.Lock()
	delete(t.cache, tenantID)
	t.mu.Unlock()
}

// Get returns the tenant's media.Store and the settings it was built from,
// constructing and caching it lazily on first use.
func (t *tenantMedia) Get(ctx context.Context, tenantID int) (media.Store, models.Settings, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if m, ok := t.cache[tenantID]; ok {
		return m.store, m.settings, nil
	}

	settings, err := t.core.GetSettings(ctx, tenantID)
	if err != nil {
		return nil, models.Settings{}, err
	}

	// Settings' JSON tags (e.g. "upload.provider") match the dotted keys
	// initSettings loads from the DB into a koanf instance at boot -
	// round-tripping through JSON here reuses that same shape for a
	// fresh, tenant-scoped koanf instance instead of duplicating the
	// DB-to-koanf loading logic (same technique as tenantMessengers).
	b, err := json.Marshal(settings)
	if err != nil {
		return nil, models.Settings{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, models.Settings{}, err
	}

	tenantKo := koanf.New(".")
	if err := tenantKo.Load(confmap.Provider(raw, "."), nil); err != nil {
		return nil, models.Settings{}, err
	}

	store, err := initMediaStore(tenantKo)
	if err != nil {
		return nil, models.Settings{}, err
	}

	m := tenantMediaStore{store: store, settings: settings}
	t.cache[tenantID] = m

	return m.store, m.settings, nil
}
