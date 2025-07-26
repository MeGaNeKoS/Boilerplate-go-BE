# Application Lifecycle (`lifecycle`)

Utilities for starting the service and shutting it down cleanly. The `Run` helper wires up signal handling, invokes the start function and closes the server listener on termination.

With the `systemd` build tag enabled, additional helpers notify systemd when the service has started.
