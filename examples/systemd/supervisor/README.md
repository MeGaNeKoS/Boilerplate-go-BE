# Supervisor Example

This directory contains a small helper used in the systemd examples. The binary listens on the `SUPERVISOR_SOCKET` Unix socket and executes `systemctl` commands to stop or restart service units.

Run it with:

```bash
go run ./examples/systemd/supervisor
```

The `connection` subpackage implements the socket protocol used by this example.
