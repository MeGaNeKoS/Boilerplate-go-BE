# Server Transports

The `server` directory contains the three transport implementations used by the
project. By default only the REST server is compiled. Build the binary with the
`grpc` or `kafka` tags to include the other transports. The entry points that
start each server live under the [`cmd`](../cmd/README.md) package.

See the [Building the Project](../README.md#building-the-project) section in the
root README for examples of using these build tags.

## REST (`server/rest`)

This folder holds the HTTP API implemented with `chi`. Routers under
`routes/` register the endpoints and call into handlers which convert requests
into service calls. These handlers are the controllers for the REST API.
Middleware provides request logging, panic recovery and context setup for
repositories and outbound clients.
The generated OpenAPI documentation is served under `/docs` when the
server starts. See the [API Documentation](../README.md#api-documentation)
section in the root README for details.

## gRPC (`server/grpc`)

The gRPC server exposes the Item service defined in the `proto` files. It uses
interceptors for panic recovery and request logging. The server converts service
errors into gRPC status codes so clients receive consistent responses.
Host and port are configured via `config.yaml` under the `GRPC` section.

## Kafka (`server/kafka`)

The Kafka consumer reads Item events from the configured topic using
`segmentio/kafka-go`. Messages are decoded into an `ItemEvent` struct and
processed similarly to REST and gRPC requests. Panics trigger a restart through
the supervisor to ensure resources are cleaned up. The broker list, topic and
consumer group are taken from the `Kafka` section of `config.yaml`.

Run all transport tests with:

```bash
go test ./server/...
```
