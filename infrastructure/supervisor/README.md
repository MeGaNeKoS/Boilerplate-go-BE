# Supervisor Helpers (`supervisor`)

This package integrates with systemd when the `systemd` build tag is enabled. It
exposes two functions:

- `RequestRestart()` notifies the supervisor to restart the main service
  specified by the `SELF_SERVICE_UNIT` environment variable. If
  `TEMP_SERVICE_UNIT` is set it is started before the restart so traffic can be
  drained from the old process.
- `OnStartUp()` stops the temporary service after the new process has started,
  unless the `TEMP_SERVICE` variable is set to `true`.

Communication happens over the Unix domain socket defined by
`SUPERVISOR_SOCKET`. When the code is built without the `systemd` tag both
functions are no-ops so the package can be imported unconditionally.

## Environment Variables

- `SUPERVISOR_SOCKET` – path to the supervisor's Unix socket
- `SELF_SERVICE_UNIT` – name of the service unit to restart
- `TEMP_SERVICE_UNIT` – optional unit started before restarting the main service
- `TEMP_SERVICE` – set to `true` for the temporary unit during a restart

See `examples/systemd` for unit files that demonstrate these settings.

Run the supervisor tests with:

```bash
go test ./infrastructure/supervisor/...
```
