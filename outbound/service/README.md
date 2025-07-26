# Service Aggregators (`outbound/service`)

Each subdirectory groups the outbound transports for a particular remote
service. A `Service` interface exposes the available transports (HTTP, gRPC,
Kafka, etc.) and constructs them on demand using shared credentials.
