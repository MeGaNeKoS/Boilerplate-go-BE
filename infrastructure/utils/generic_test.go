package utils

import "testing"

func TestBoolToInt(t *testing.T) {
	if got := boolToInt(true); got != 1 {
		t.Errorf("boolToInt(true)=%d, want 1", got)
	}
	if got := boolToInt(false); got != 0 {
		t.Errorf("boolToInt(false)=%d, want 0", got)
	}
}

func TestReplacePlaceholders(t *testing.T) {
	input := "Hello ${name}, cost is $${price} and ${unknown}!"
	values := map[string]string{"name": "Bob", "unknown": "X"}
	got := ReplacePlaceholders(input, values)
	want := "Hello Bob, cost is ${price} and X!"
	if got != want {
		t.Errorf("ReplacePlaceholders()=%q, want %q", got, want)
	}

	if ReplacePlaceholders("${missing", nil) != "${missing" {
		t.Errorf("unterminated placeholder not handled")
	}

	if ReplacePlaceholders("${absent}", nil) != "${absent}" {
		t.Errorf("missing key placeholder changed")
	}
}
