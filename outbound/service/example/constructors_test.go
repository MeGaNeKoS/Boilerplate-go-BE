package example

import "testing"

func TestNewExampleGRPCOutbound(t *testing.T) {
	l := stubLogger{parent: "p"}
	ob := NewExampleGRPCOutbound(l)
	eob, ok := ob.(*exampleGRPCOutbound)
	if !ok {
		t.Fatalf("expected *exampleGRPCOutbound, got %T", ob)
	}
	if eob.log != l {
		t.Fatalf("unexpected fields: %#v", eob)
	}
}

func TestNewExampleOutbound(t *testing.T) {
	l := stubLogger{parent: "p"}
	ob := NewExampleOutbound(l)
	eob, ok := ob.(*exampleOutbound)
	if !ok {
		t.Fatalf("expected *exampleOutbound, got %T", ob)
	}
	if eob.log != l {
		t.Fatalf("unexpected fields: %#v", eob)
	}
}
