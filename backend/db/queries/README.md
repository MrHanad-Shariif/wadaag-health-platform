# sqlc queries

`sqlc generate` (config in `../../sqlc.yaml`) reads `.sql` files from this
directory plus the schema in `../migrations`, and generates type-safe Go
into `internal/platform/sqlcgen`. Regenerate after adding or changing a
query or migration:

```
sqlc generate
```

Each module's repository (e.g. `internal/identity/repository.go`) wraps the
generated `Queries` methods and converts pgx's `pgtype.*` wire types to the
plain Go types used in the module's domain model.
