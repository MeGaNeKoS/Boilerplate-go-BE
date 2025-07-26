# Shared Packages (`pkg`)

This folder contains small helper libraries that are reused across the project.
All code here is infrastructure agnostic so it can be imported from the service layer,
server or other packages without creating circular dependencies.

## `code`

Defines the `code.Code` struct used for centralised error handling. Services
return these values instead of raw errors so the server layer can map
them directly to HTTP status codes, gRPC errors or Kafka codes.

## `lifecycle`

Utilities for starting and stopping the application gracefully. The `Run`
function wraps the main `Start` function, wiring up signal handling and calling
the registered shutdown hooks. When built with the `systemd` tag it also notifies
systemd when the service is ready.

## `logger`

A lightweight structured logger that writes to level based log files and can also
stream logs to stdout. It is configured through `config.LogConfig` and is safe
for concurrent use by multiple goroutines.


## `validator`

Thin wrapper around `go-playground/validator` providing `ValidateStruct` and
custom date validation rules.

Run all package tests with:

```bash
go test ./pkg/...
```
