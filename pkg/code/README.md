# Error Codes (`code`)

This package defines the `Code` struct used to represent application errors. Each constant maps an internal error to HTTP and gRPC status codes. Kafka errors use their own `KafkaCode` strings defined in `infrastructure/enums`, so transports can forward errors without extra mapping.

Returning a `Code` from services means controllers do not need additional error mapping—the transport layer simply forwards the correct status. Changing from REST to gRPC or Kafka does not require rewriting controller logic.
