# dbmigrator

A thin wrapper around [golang-migrate](https://github.com/golang-migrate/migrate) that runs PostgreSQL schema migrations from one or more embedded filesystems (`embed.FS`).

## Features

- Runs `UP` migrations automatically on call
- Reads migration files from `embed.FS` — no filesystem access at runtime
- Supports **multiple sources** merged into a single migration sequence
- Returns the schema version before and after migration
- Detects and rejects dirty database state before running

## Migration file format

Files follow the standard golang-migrate naming convention:

```
{version}_{title}.up.sql
{version}_{title}.down.sql
```

- `version` — positive integer, determines execution order
- `title` — arbitrary label (underscores, letters, digits)
- Both `.up.sql` and `.down.sql` files are required for each version

Example:

```
migrations/
  001_create_users.up.sql
  001_create_users.down.sql
  002_add_email_index.up.sql
  002_add_email_index.down.sql
```

Version numbers must be **globally unique** across all sources when using multiple sources.

## Usage

### Single source

```go
package main

import (
    "embed"
    "log"

    "github.com/ashep/go-app/dbmigrator"
    "github.com/rs/zerolog"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
    logger := zerolog.New(os.Stderr)

    result, err := dbmigrator.RunPostgres(
        "postgres://user:pass@localhost:5432/mydb",
        logger,
        dbmigrator.Source{FS: migrationsFS, Path: "migrations"},
    )
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("migrated: v%d -> v%d", result.PrevVersion, result.NewVersion)
}
```

### Multiple sources

Useful when migrations are spread across multiple packages (e.g., a core module and a plugin each own their own schema files).

```go
//go:embed core/migrations/*.sql
var coreFS embed.FS

//go:embed plugin/migrations/*.sql
var pluginFS embed.FS

result, err := dbmigrator.RunPostgres(
    "postgres://user:pass@localhost:5432/mydb",
    logger,
    dbmigrator.Source{FS: coreFS,   Path: "core/migrations"},
    dbmigrator.Source{FS: pluginFS, Path: "plugin/migrations"},
)
```

Sources are merged in the order they are provided. If two sources contain a file with the same name, the first source wins and the duplicate is silently ignored. To avoid ambiguity, keep version numbers unique across all sources.

## API

### `RunPostgres`

```go
func RunPostgres(url string, l zerolog.Logger, sources ...Source) (*MigrationResult, error)
```

Connects to the PostgreSQL database at `url`, merges all provided `sources`, and applies any pending `UP` migrations.

Returns `MigrationResult` on success, or an error if:
- A source path cannot be read
- The database connection fails
- The current schema version is dirty
- A migration fails to apply

Calling `RunPostgres` when the database is already up-to-date is safe — it returns the current version without error.

### `Source`

```go
type Source struct {
    FS   embed.FS
    Path string
}
```

Pairs an embedded filesystem with the subdirectory within it that contains migration files.

### `MigrationResult`

```go
type MigrationResult struct {
    PrevVersion uint
    NewVersion  uint
}
```

Both fields are `0` when the database has no migrations applied yet (`ErrNilVersion` is treated as version 0).

## Error handling

| Situation | Behavior |
|---|---|
| Database already at latest version | Returns current version, `nil` error |
| Database has no migrations applied | `PrevVersion: 0`, runs all migrations |
| Dirty version detected | Returns error: `current version is dirty: N` |
| Duplicate file name across sources | First source wins; duplicate is ignored |
| Duplicate version number across sources | golang-migrate returns an error |
| Migration SQL fails | Returns error, database left in dirty state |

## Schema migrations table

golang-migrate tracks the applied version in a `schema_migrations` table created automatically in the target database. Do not modify this table manually.

## Dependencies

- [`github.com/golang-migrate/migrate/v4`](https://github.com/golang-migrate/migrate)
- [`github.com/golang-migrate/migrate/v4/database/pgx/v5`](https://github.com/golang-migrate/migrate/tree/master/database/pgx/v5)
- [`github.com/rs/zerolog`](https://github.com/rs/zerolog)
