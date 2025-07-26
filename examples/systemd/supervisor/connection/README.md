# Supervisor Connection

Implements the Unix socket listener used by the supervisor example. The `Listen` function accepts commands and performs `systemctl` actions like stopping or restarting services.

Tests verify the command parsing and error handling.
