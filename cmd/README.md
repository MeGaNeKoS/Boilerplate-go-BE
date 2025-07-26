# Entry points (`cmd` package)

The `cmd` directory contains the executable entry points for this service. Each
file exposes a `Start` function that is called from `main.go` via the lifecycle
runner. The function loads configuration, initialises shared dependencies and
then launches the appropriate server.

## Files

- **`start_rest.go`** – starts the REST HTTP API (default build)
- **`start_grpc.go`** – included when building with `-tags grpc` to launch the
  gRPC server
- **`start_kafka.go`** – included when building with `-tags kafka` to run the
  Kafka consumer
- **`cmd.init.go`** – shared helpers for flag parsing and dependency setup

## Flags

All entry points understand the following command line flags:

```
-v, --version        Print build version information and exit
--config <path>      Path to the configuration file (default `config.yaml`)
```

Example usage:

```bash
go run . --config custom.yaml

go run -tags grpc . -v
```

## Systemd Environment Variables

When running under systemd, the REST server honours
`SYSTEMD_SOCKET_ACTIVATION=true` to enable socket activation. The optional
supervisor uses these variables to coordinate restarts:

- `SUPERVISOR_SOCKET` – path to the supervisor's Unix socket
- `TEMP_SERVICE_UNIT` – unit name started before restarting the main service
- `SELF_SERVICE_UNIT` – name of the current service unit
- `TEMP_SERVICE` – set to `true` for the temporary unit during a restart

See `examples/systemd` for working unit files that demonstrate these settings.
