package enums

import "testing"

func TestKafkaCodeStrings(t *testing.T) {
	cases := map[KafkaCode]string{
		KafkaInvalidArgument:   "INVALID_ARGUMENT",
		KafkaUnauthenticated:   "UNAUTHENTICATED",
		KafkaPermissionDenied:  "PERMISSION_DENIED",
		KafkaNotFound:          "NOT_FOUND",
		KafkaUnimplemented:     "UNIMPLEMENTED",
		KafkaAborted:           "ABORTED",
		KafkaResourceExhausted: "RESOURCE_EXHAUSTED",
		KafkaInternal:          "INTERNAL",
		KafkaUnavailable:       "UNAVAILABLE",
		KafkaDeadlineExceeded:  "DEADLINE_EXCEEDED",
	}
	for c, exp := range cases {
		if string(c) != exp {
			t.Fatalf("expected %s got %s", exp, c)
		}
	}
}
