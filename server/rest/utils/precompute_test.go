package utils

import (
	"reflect"
	"testing"
)

func TestPrecomputeErrorCodes(t *testing.T) {
	names := PrecomputeErrorCodes(modulePath+"/server/rest/handlers", "GetItem")
	expected := []string{"ErrInternalServerError", "ErrItemNotFound"}
	if !reflect.DeepEqual(names, expected) {
		t.Fatalf("expected %v, got %v", expected, names)
	}
}

func TestPrecomputeErrorCodesMissing(t *testing.T) {
	names := PrecomputeErrorCodes(modulePath+"/server/rest/handlers", "NoSuchFunc")
	if len(names) != 0 {
		t.Fatalf("expected no codes, got %v", names)
	}
}
