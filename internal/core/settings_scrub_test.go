package core

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"

	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/models"
)

// testPool connects as the admin/table-owning role (adminDSN, defined in
// tenant_test.go) -- these tests write settings rows directly and need
// DML rights RLS-bound tenant-app roles don't have.
func testPool(t *testing.T) *sqlx.DB {
	t.Helper()
	pool, err := sqlx.Connect("postgres", adminDSN())
	if err != nil {
		t.Skipf("skipping: no reachable test database: %v", err)
	}
	if err := pool.Ping(); err != nil {
		pool.Close()
		t.Skipf("skipping: test database not reachable: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

// testCoreForSettings builds a minimal *Core with just the three settings
// statements prepared directly from their known SQL text (queries/misc.sql)
// -- there's no existing precedent in this package for loading the full
// models.Queries set via goyesql in a test (the sole existing test,
// tenant_test.go, avoids Core.q entirely by using raw SQL), and hand-
// preparing three statements is far less machinery than reproducing the
// boot-time query-loading pipeline just for this.
func testCoreForSettings(t *testing.T, pool *sqlx.DB) *Core {
	t.Helper()

	getSettings, err := pool.Preparex(`SELECT JSON_OBJECT_AGG(key, value) AS settings FROM (SELECT * FROM settings WHERE tenant_id = $1 ORDER BY key) t`)
	if err != nil {
		t.Fatalf("prepare get-settings: %v", err)
	}
	updateSettings, err := pool.Preparex(`UPDATE settings AS s SET value = c.value FROM(SELECT * FROM JSONB_EACH($1)) AS c(key, value) WHERE s.key = c.key AND s.tenant_id = $2`)
	if err != nil {
		t.Fatalf("prepare update-settings: %v", err)
	}
	updateSettingsByKey, err := pool.Preparex(`UPDATE settings SET value = $2, updated_at = NOW() WHERE key = $1 AND tenant_id = $3`)
	if err != nil {
		t.Fatalf("prepare update-settings-by-key: %v", err)
	}
	t.Cleanup(func() {
		getSettings.Close()
		updateSettings.Close()
		updateSettingsByKey.Close()
	})

	lang, err := i18n.New([]byte(`{"_.code":"en","_.name":"English"}`))
	if err != nil {
		t.Fatalf("building i18n: %v", err)
	}

	return &Core{
		db: pool,
		q: &models.Queries{
			GetSettings:         getSettings,
			UpdateSettings:      updateSettings,
			UpdateSettingsByKey: updateSettingsByKey,
		},
		i18n: lang,
	}
}

func TestScrubSettings_ManagedByPlatformLock(t *testing.T) {
	pool := testPool(t)
	c := testCoreForSettings(t, pool)

	var tenantID int
	if err := pool.Get(&tenantID, `INSERT INTO tenants (slug, name) VALUES ($1, 'Scrub Lock Test') RETURNING id`,
		fmt.Sprintf("scrub-lock-test-%d", time.Now().UnixNano())); err != nil {
		t.Fatalf("inserting test tenant: %v", err)
	}
	t.Cleanup(func() {
		pool.MustExec(`DELETE FROM tenants WHERE id = $1`, tenantID)
	})

	locked := map[string]any{
		"enabled": true, "url": "https://platform.example", "api_key": "platform-key",
		"integration_id": "platform-uuid", "managed_by_platform": true,
	}
	b, _ := json.Marshal(locked)
	pool.MustExec(`INSERT INTO settings (tenant_id, key, value) VALUES ($1, 'scrub', $2::jsonb) ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value`, tenantID, b)

	ctx := context.Background()

	t.Run("UpdateSettings can't change a locked scrub row", func(t *testing.T) {
		s, err := c.GetSettings(ctx, tenantID)
		if err != nil {
			t.Fatalf("GetSettings: %v", err)
		}
		if !s.Scrub.ManagedByPlatform || s.Scrub.URL != "https://platform.example" {
			t.Fatalf("unexpected starting state: %+v", s.Scrub)
		}

		// A tenant admin's own full-object save trying to flip everything.
		s.Scrub.Enabled = false
		s.Scrub.URL = "https://evil.example"
		s.Scrub.APIKey = "stolen-key"
		s.Scrub.IntegrationID = "attacker-uuid"
		s.Scrub.ManagedByPlatform = false
		if err := c.UpdateSettings(ctx, tenantID, s); err != nil {
			t.Fatalf("UpdateSettings: %v", err)
		}

		after, err := c.GetSettings(ctx, tenantID)
		if err != nil {
			t.Fatalf("GetSettings after: %v", err)
		}
		if after.Scrub.URL != "https://platform.example" || !after.Scrub.ManagedByPlatform {
			t.Errorf("locked scrub config was changed by UpdateSettings: %+v", after.Scrub)
		}
	})

	t.Run("UpdateSettingsByKey refuses a locked scrub row", func(t *testing.T) {
		bypass, _ := json.Marshal(map[string]any{
			"enabled": true, "url": "https://evil.example", "api_key": "x",
			"integration_id": "y", "managed_by_platform": false,
		})
		err := c.UpdateSettingsByKey(ctx, tenantID, "scrub", bypass)
		if err == nil {
			t.Fatal("expected an error locking out the write, got nil")
		}
		if he, ok := err.(*echo.HTTPError); !ok || he.Code != 403 {
			t.Errorf("error = %v, want a 403 echo.HTTPError", err)
		}

		after, err := c.GetSettings(ctx, tenantID)
		if err != nil {
			t.Fatalf("GetSettings after: %v", err)
		}
		if after.Scrub.URL != "https://platform.example" {
			t.Errorf("locked scrub config was changed by UpdateSettingsByKey: %+v", after.Scrub)
		}
	})

	t.Run("unmanaged tenant's scrub row is unaffected", func(t *testing.T) {
		var otherTenantID int
		if err := pool.Get(&otherTenantID, `INSERT INTO tenants (slug, name) VALUES ($1, 'Scrub Unmanaged Test') RETURNING id`,
			fmt.Sprintf("scrub-unmanaged-test-%d", time.Now().UnixNano())); err != nil {
			t.Fatalf("inserting test tenant: %v", err)
		}
		t.Cleanup(func() {
			pool.MustExec(`DELETE FROM tenants WHERE id = $1`, otherTenantID)
		})

		unmanaged, _ := json.Marshal(map[string]any{
			"enabled": false, "url": "", "api_key": "", "integration_id": "", "managed_by_platform": false,
		})
		pool.MustExec(`INSERT INTO settings (tenant_id, key, value) VALUES ($1, 'scrub', $2::jsonb) ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value`, otherTenantID, unmanaged)

		want, _ := json.Marshal(map[string]any{
			"enabled": true, "url": "https://own.example", "api_key": "own-key",
			"integration_id": "own-id", "managed_by_platform": false,
		})
		if err := c.UpdateSettingsByKey(ctx, otherTenantID, "scrub", want); err != nil {
			t.Fatalf("UpdateSettingsByKey on an unmanaged tenant should succeed: %v", err)
		}

		after, err := c.GetSettings(ctx, otherTenantID)
		if err != nil {
			t.Fatalf("GetSettings: %v", err)
		}
		if after.Scrub.URL != "https://own.example" {
			t.Errorf("unmanaged tenant's own settings save didn't take effect: %+v", after.Scrub)
		}
	})
}
