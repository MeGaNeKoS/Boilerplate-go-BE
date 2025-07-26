package enums

// KafkaCode enumerates error codes used when returning Kafka responses.
type KafkaCode string

const (
	KafkaInvalidArgument   KafkaCode = "INVALID_ARGUMENT"
	KafkaUnauthenticated   KafkaCode = "UNAUTHENTICATED"
	KafkaPermissionDenied  KafkaCode = "PERMISSION_DENIED"
	KafkaNotFound          KafkaCode = "NOT_FOUND"
	KafkaUnimplemented     KafkaCode = "UNIMPLEMENTED"
	KafkaAborted           KafkaCode = "ABORTED"
	KafkaResourceExhausted KafkaCode = "RESOURCE_EXHAUSTED"
	KafkaInternal          KafkaCode = "INTERNAL"
	KafkaUnavailable       KafkaCode = "UNAVAILABLE"
	KafkaDeadlineExceeded  KafkaCode = "DEADLINE_EXCEEDED"
)
