package core

import (
	"context"
	"io"
	"log"
	"testing"
)

// Whether new subscribers get an opt-in confirmation was read from tenant
// 1's settings at boot, so a workspace turning it off still sent them (or
// turning it on sent none).
func TestSendsOptinConfirmation_PerWorkspace(t *testing.T) {
	pool := testPool(t)
	c := testCoreForSettings(t, pool)
	c.log = log.New(io.Discard, "", 0)
	c.consts.SendOptinConfirmation = true // tenant 1's, at boot

	off := newListFixture(t, pool)
	pool.MustExec(`INSERT INTO settings (tenant_id, key, value) VALUES ($1, 'app.send_optin_confirmation', 'false')`, off.tenant)
	on := newListFixture(t, pool)
	pool.MustExec(`INSERT INTO settings (tenant_id, key, value) VALUES ($1, 'app.send_optin_confirmation', 'true')`, on.tenant)

	ctx := context.Background()
	if c.sendsOptinConfirmation(ctx, off.tenant) {
		t.Error("a workspace with opt-in confirmations off would still send them")
	}
	if !c.sendsOptinConfirmation(ctx, on.tenant) {
		t.Error("a workspace with opt-in confirmations on wouldn't send them")
	}
}
