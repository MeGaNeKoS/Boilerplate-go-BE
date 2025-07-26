# Enumerations (`enums`)

This directory groups common enumerations used throughout the infrastructure layer. Each enum is defined as its own type for strong typing and easy logging or JSON marshalling.

Enumerations typically use string constants so log output remains readable. Add new enums in separate files or extend existing ones when needed.

The folder currently includes:

- `DirtyStrategy` – controls how database migrations handle dirty states.
- `KafkaCode` – names of error codes used in Kafka responses.

Run all enum tests with:

```bash
go test ./infrastructure/enums/...
```
