package supervisor

import "testing"

func TestRequestRestartNoop(t *testing.T) {
	if err := RequestRestart(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestOnStartUpNoop(t *testing.T) {
	if err := OnStartUp(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
