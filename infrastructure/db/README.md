# Database Helpers (`db`)

This package wraps the standard `database/sql` API and is responsible for managing the MySQL connection configured in `config.Cfg`. It exposes a singleton via `GetDBInstance` so callers can share a pooled connection. Queries can be executed directly with `ExecContext` or run inside transactions using `WithTransaction`.

The helper also provides utility methods such as `GetGoquDialect` for generating SQL using the [`goqu`](https://github.com/doug-martin/goqu) library and `Ping` for health checks. Tests under this directory rely on `sqlmock` to simulate database behaviour without requiring a running MySQL server.

During startup the entry points call `RunMigrations` with the path specified in `Database.MigrationPath` (default `file://migrations`) to apply database migrations using [golang-migrate](https://github.com/golang-migrate/migrate). Dirty migrations are handled according to the `DirtyStrategy` enum set in `Database.DirtyStrategy` (`retry` or `skip`). When retrying, the failed migration is automatically rolled back using its `*.down.sql` file before applying migrations again.
Update the `Database` section of `config.yaml` with your connection details.
See the **Database Migrations** section in the root [README.md](../../README.md) for the full workflow.

Run all database tests with:

```bash
go test ./infrastructure/db/...
```
