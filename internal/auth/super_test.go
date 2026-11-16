package auth

import "testing"

// Super Admin was recognised by role ID 1, which only tenant 1's role has,
// so every other workspace's Super Admin lacked any permission added after
// its role was created.
func TestSuperAdminHasEveryPermission(t *testing.T) {
	super := User{PermissionsMap: map[string]struct{}{}}
	super.UserRole.ID = 41
	super.UserRole.Super = true
	if !super.HasPerm("apikeys:manage") || !super.IsSuperAdmin() {
		t.Error("a workspace's Super Admin (role 41) lacks a permission its role doesn't list")
	}
	if all, ids := super.GetPermittedLists(PermTypeGet | PermTypeManage); !all || ids != nil {
		t.Errorf("GetPermittedLists = %v, %v; want every list", all, ids)
	}

	other := User{PermissionsMap: map[string]struct{}{"campaigns:get": {}}}
	other.UserRole.ID = 1
	if other.HasPerm("apikeys:manage") || other.IsSuperAdmin() {
		t.Error("a non-super role with ID 1 is treated as Super Admin")
	}
}
