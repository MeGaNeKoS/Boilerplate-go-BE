package system

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bouk/monkey"
)

func TestServiceEcho(t *testing.T) {
	svc := NewService()
	resp, code := svc.Echo(context.Background())
	if code != nil || resp != "echo" {
		t.Fatalf("unexpected resp %#v code %#v", resp, code)
	}
}

func TestServiceCrash(t *testing.T) {
	svc := NewService()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		} else if r != "intentional crash" {
			t.Fatalf("unexpected panic %v", r)
		}
	}()
	svc.Crash(context.Background())
}

func TestServiceLong(t *testing.T) {
	svc := NewService()

	var slept bool
	patch := monkey.Patch(time.Sleep, func(time.Duration) { slept = true })
	defer patch.Unpatch()

	resp, code := svc.Long(context.Background(), 1)
	pid := os.Getpid()
	expected := fmt.Sprintf("%d", pid)
	if code != nil || resp != expected || !slept {
		t.Fatalf("unexpected resp %#v code %#v slept %v", resp, code, slept)
	}

	slept = false
	resp, code = svc.Long(context.Background(), 0)
	if code != nil || resp != expected {
		t.Fatalf("unexpected resp %#v code %#v", resp, code)
	}
	if slept {
		t.Fatalf("sleep should not have been called")
	}
}
