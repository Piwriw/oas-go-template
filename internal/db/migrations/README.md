# SQL migrations

Add every schema change as a timestamped up/down pair:

```text
YYYYMMDDHHMMSS_description.up.sql
YYYYMMDDHHMMSS_description.down.sql
```

For example:

```text
20260801143000_create_users.up.sql
20260801143000_create_users.down.sql
```

The files are embedded into the server binary. On startup, `golang-migrate`
applies pending `up` files in version order and records the current version in
`schema_migrations`. Never edit or reuse an applied version. The `down` file
must reverse its matching `up` file. Write SQL for the database selected by
`db.driver`; a project should use one production database dialect consistently.
