# Database Schema

This folder contains canonical table definitions used to generate migrations.
Developers edit these files when updating the data model and then run a tool to
diff them against the current database and create new migration scripts.

The directory currently contains:

- `item.sql` – creates the `items` table and seeds a few example rows.
- `item_detail.sql` – defines an `item_details` table referencing `items`.


For example using [Atlas](https://atlasgo.io/):

```bash
atlas schema apply --to file://schema --dev-url "mysql://user:pass@localhost/db"
atlas migrate diff --dir schema --dir migrations --name add_table
```

The generated migration files will be placed under `migrations` and applied
automatically on startup. See the [Database Migrations](../README.md#database-migrations)
section in the root README for the full workflow.
