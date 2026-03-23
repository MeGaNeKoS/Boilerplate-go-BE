package dto

import "testing"

func TestRootPermissionActions(t *testing.T) {
	rp := RootPermission{Action: []string{"create", "read", "update", "delete"}}
	if !rp.hasAction("create") || !rp.hasAction("update") {
		t.Fatalf("hasAction failed")
	}
	if rp.CanCreateItem() != true {
		t.Fatalf("CanCreateItem failed")
	}
	if !rp.CanReadItem() {
		t.Fatalf("CanReadItem failed")
	}
	if !rp.CanUpdateItem() {
		t.Fatalf("CanUpdateItem failed")
	}
	if !rp.CanDeleteItem() {
		t.Fatalf("CanDeleteItem failed")
	}
	rp = RootPermission{Action: []string{"read"}}
	if rp.CanCreateItem() || rp.CanUpdateItem() || rp.CanDeleteItem() {
		t.Fatalf("permissions incorrectly granted")
	}
}

func TestGetEnvironment(t *testing.T) {
	user := &JWTUser{Environment: "production"}
	if got := user.GetEnvironment(); got != "production" {
		t.Fatalf("expected production, got %s", got)
	}

	user2 := &JWTUser{Environment: ""}
	if got := user2.GetEnvironment(); got != "" {
		t.Fatalf("expected empty string, got %s", got)
	}
}

func TestHasActionNotFound(t *testing.T) {
	rp := RootPermission{Action: []string{"read"}}
	if rp.hasAction("missing") {
		t.Fatalf("hasAction should return false for missing action")
	}
}

func TestEmptyPermissions(t *testing.T) {
	rp := RootPermission{}
	if rp.hasAction("read") {
		t.Fatalf("hasAction should return false for empty actions")
	}
	if rp.CanCreateItem() || rp.CanReadItem() || rp.CanUpdateItem() || rp.CanDeleteItem() {
		t.Fatalf("all permissions should be false with empty actions")
	}
}
