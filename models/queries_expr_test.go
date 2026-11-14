package models

import (
	"errors"
	"testing"
)

// A subscriber query expression runs inside the tenant's transaction; one
// that changes settings could switch tenants mid-query.
func TestCheckQueryExpr(t *testing.T) {
	for _, bad := range []string{
		`set_config('app.current_tenant', '', true) IS NOT NULL`,
		`pg_catalog.SET_CONFIG('app.all_tenants', 'on', true) = 'on'`,
		`"set_config"('app.current_tenant', '', true) IS NOT NULL`,
		`U&"\\0073et_config"('a', 'b', true) IS NOT NULL`,
	} {
		if err := CheckQueryExpr(bad); !errors.Is(err, ErrQueryExprNotAllowed) {
			t.Errorf("CheckQueryExpr(%q) = %v, want ErrQueryExprNotAllowed", bad, err)
		}
	}
	for _, ok := range []string{"", `subscribers.attribs->>'city' = 'Harare'`, `subscribers.email LIKE '%@config.example'`} {
		if err := CheckQueryExpr(ok); err != nil {
			t.Errorf("CheckQueryExpr(%q) = %v, want allowed", ok, err)
		}
	}
}
