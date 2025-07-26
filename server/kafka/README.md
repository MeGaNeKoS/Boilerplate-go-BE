# Kafka Consumer (`server/kafka`)

This package provides the Kafka transport used by the project. It consumes `ItemEvent` messages from the configured topic using `segmentio/kafka-go`. The consumer builds the same service layer as the REST and gRPC servers and processes each message in `handleMessage`.

Service errors include a `KafkaCode` string defined in `infrastructure/enums`. Handlers can forward this code in reply messages if required.

Panics are recovered and trigger a supervisor restart so the consumer can restart cleanly.
