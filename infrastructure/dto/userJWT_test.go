package dto

import "testing"

func TestRootPermissionActions(t *testing.T) {
	rp := RootPermission{Action: []string{"create", "read", "update", "delete"}}
	if !rp.hasAction("create") || !rp.hasAction("update") {
		t.Fatalf("hasAction failed")
	}
	if rp.CanCreateTicket() != true {
		t.Fatalf("CanCreateTicket failed")
	}
	if !rp.CanReadTicket() {
		t.Fatalf("CanReadTicket failed")
	}
	if !rp.CanUpdateTicket() {
		t.Fatalf("CanUpdateTicket failed")
	}
	if !rp.CanDeleteTicket() {
		t.Fatalf("CanDeleteTicket failed")
	}
	rp = RootPermission{Action: []string{"read"}}
	if rp.CanCreateTicket() || rp.CanUpdateTicket() || rp.CanDeleteTicket() {
		t.Fatalf("permissions incorrectly granted")
	}
}
