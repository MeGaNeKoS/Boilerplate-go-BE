# Infrastructure Layer

This directory holds implementations used by the service layer and the server transports. It contains database helpers and other utilities.


The design overview in the root [README.md](../README.md) explains how these pieces fit into the overall architecture.

## `config`

Loads YAML settings into `config.Cfg`. Entry points under `cmd` use this package to configure the service. See `config/README.md` for usage details.

## `db`

Initialises the MySQL connection based on `config.Cfg` and exposes methods for running queries and transactions. Tests use `sqlmock` to verify behaviour without a real database. See `db/README.md` for package usage.


Outbound clients now live under the top-level `outbound` directory.

Grouped clients for calling external services. The example implementation demonstrates how to build REST, gRPC and Kafka clients and cache them using the aggregator.

## `dto`

Structs passed between layers. Domain specific DTOs live in
subdirectories (for example `dto/item`), while shared types like
`JWTUser` or HTTP `response` helpers sit at the package root. See
`dto/README.md` for details.
## `enums`

Common enumerations shared across the infrastructure layer. The folder is flat - each file collects related constants. See `enums/README.md` for guidelines.


## `supervisor`

Restart helpers used when running under systemd. `RequestRestart` and `OnStartUp`
communicate with the supervisor over a Unix socket to coordinate restarts of the
main service and any temporary unit. The functions are no-ops without the
`systemd` build tag. See `supervisor/README.md` for environment variables and
usage details.

## `utils`

Miscellaneous helpers such as context storage, HTTP response builders and a JWT service and path utilities. These utilities are transport agnostic and may be imported by any layer. See `utils/README.md` for details.

## `validator`

Validation helpers have moved to `pkg/validator`. The package wraps
`go-playground/validator` with custom `notBeforeNow` and `notAfterNow` rules.

Run all infrastructure tests with:

```bash
go test ./infrastructure/...
```

