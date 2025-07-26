# Structured Logger (`logger`)

Provides a lightweight logger with support for log levels, file rotation and optional streaming to stdout. Configure it using `config.LogConfig` and create new instances with `NewLogger`.

The package is safe for concurrent use and allows custom watchers to react to log events.
