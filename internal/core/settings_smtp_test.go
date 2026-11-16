package core

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/nerd-zero/listnun/models"
)

// A workspace admin's save must not change platform-managed SMTP: a server
// whose host changed but whose password was left blank would otherwise
// keep its stored credentials and use them against the new host.
func TestSMTPSettings_ManagedByPlatformLock(t *testing.T) {
	pool := testPool(t)
	c := testCoreForSettings(t, pool)
	ctx := context.Background()

	newTenant := func(name string, smtp []models.SMTPServer) int {
		var id int
		if err := pool.Get(&id, `INSERT INTO tenants (slug, name) VALUES ($1, $2) RETURNING id`,
			fmt.Sprintf("smtp-lock-%d", time.Now().UnixNano()), name); err != nil {
			t.Fatalf("inserting tenant: %v", err)
		}
		t.Cleanup(func() { pool.MustExec(`DELETE FROM tenants WHERE id = $1`, id) })
		b, _ := json.Marshal(smtp)
		pool.MustExec(`INSERT INTO settings (tenant_id, key, value) VALUES ($1, 'smtp', $2::jsonb)
			ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value`, id, b)
		return id
	}

	platform := models.SMTPServer{
		Name: "email-postmark", UUID: "pm-1", Enabled: true, Host: "smtp.postmarkapp.com", Port: 587,
		AuthProtocol: "plain", Username: "server-token", Password: "server-token", ManagedByPlatform: true,
	}
	managedID := newTenant("Managed", []models.SMTPServer{platform})

	t.Run("a full save keeps the platform's servers", func(t *testing.T) {
		s, err := c.GetSettings(ctx, managedID)
		if err != nil {
			t.Fatalf("GetSettings: %v", err)
		}
		s.SMTP[0].Host = "smtp.attacker.example"
		s.SMTP[0].Password = ""
		s.SMTP[0].ManagedByPlatform = false
		s.SMTP = append(s.SMTP, models.SMTPServer{Name: "email-own", Enabled: true, Host: "smtp.own.example"})
		if err := c.UpdateSettings(ctx, managedID, s); err != nil {
			t.Fatalf("UpdateSettings: %v", err)
		}

		after, err := c.GetSettings(ctx, managedID)
		if err != nil {
			t.Fatalf("GetSettings: %v", err)
		}
		if len(after.SMTP) != 1 || !reflect.DeepEqual(after.SMTP[0], platform) {
			t.Errorf("SMTP after a workspace save = %+v, want the platform's server unchanged", after.SMTP)
		}
	})

	t.Run("a save of the smtp key alone is refused", func(t *testing.T) {
		b, _ := json.Marshal([]models.SMTPServer{{Name: "email-own", Enabled: true, Host: "smtp.attacker.example", UUID: "pm-1"}})
		err := c.UpdateSettingsByKey(ctx, managedID, "smtp", b)
		if he, ok := err.(*echo.HTTPError); !ok || he.Code != 403 {
			t.Fatalf("error = %v, want a 403", err)
		}
	})

	t.Run("a workspace's own servers can still be changed", func(t *testing.T) {
		ownID := newTenant("Own", []models.SMTPServer{{Name: "email-own", UUID: "o-1", Enabled: true, Host: "smtp.own.example"}})
		s, err := c.GetSettings(ctx, ownID)
		if err != nil {
			t.Fatalf("GetSettings: %v", err)
		}
		s.SMTP[0].Host = "smtp.new.example"
		if err := c.UpdateSettings(ctx, ownID, s); err != nil {
			t.Fatalf("UpdateSettings: %v", err)
		}
		after, _ := c.GetSettings(ctx, ownID)
		if after.SMTP[0].Host != "smtp.new.example" {
			t.Errorf("host = %q, want the workspace's own change saved", after.SMTP[0].Host)
		}
	})
}
