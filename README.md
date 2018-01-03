This is a fork of https://bitbucket.org/liamstask/goose/src with SQLite support removed in order 
to speedup builds and tests with direct or transitive dependencies on Goose

Added new goose commands:
 1. create_db [soft]
 2. drop_db [soft]

# goose

goose is a database migration tool.

You can manage your database's evolution by creating incremental SQL or Go scripts.

# Install

    $ go get github.com/gojuno/goose

This will install the `goose` binary to your `$GOPATH/bin` directory.

You can also build goose into your own applications by importing `github.com/gojuno/goose/lib/goose`. 
Documentation is available at [godoc.org](http://godoc.org/bitbucket.org/liamstask/goose/lib/goose).

NOTE: the API is still new, and may undergo some changes.

# Usage

goose provides several commands to help manage your database schema.

## create

Create a new Go migration.

    $ goose create AddSomeColumns
    $ goose: created db/migrations/20130106093224_AddSomeColumns.go

Edit the newly created script to define the behavior of your migration.

You can also create an SQL migration:

    $ goose create AddSomeColumns sql
    $ goose: created db/migrations/20130106093224_AddSomeColumns.sql

## up

Apply all available migrations.

    $ goose up
    $ goose: migrating db environment 'development', current version: 0, target: 3
    $ OK    001_basics.sql
    $ OK    002_next.sql
    $ OK    003_and_again.go

## up-to

Migrate up to a specific version.

    $ goose up-to 20170506082420
    $ OK    20170506082420_create_table.sql

## down

Roll back a single migration from the current version.

    $ goose down
    $ goose: migrating db environment 'development', current version: 3, target: 2
    $ OK    003_and_again.go

## down-to

Roll back migrations to a specific version.

    $ goose down-to 20170506082527
    $ OK    20170506082527_alter_column.sql

## redo

Roll back the most recently applied migration, then run it again.

    $ goose redo
    $ goose: migrating db environment 'development', current version: 3, target: 2
    $ OK    003_and_again.go
    $ goose: migrating db environment 'development', current version: 2, target: 3
    $ OK    003_and_again.go

## status

Print the status of all migrations:

    $ goose status
    $ goose: status for environment 'development'
    $   Applied At                  Migration
    $   =======================================
    $   Sun Jan  6 11:25:03 2013 -- 001_basics.sql
    $   Sun Jan  6 11:25:03 2013 -- 002_next.sql
    $   Pending                  -- 003_and_again.go

Note: for MySQL [parseTime flag](https://github.com/go-sql-driver/mysql#parsetime) must be enabled.

## version

Print the current version of the database:

    $ goose version
    $ goose: version 002

# Migrations

goose supports migrations written in SQL or in Go.

## SQL Migrations

A sample SQL migration looks like:

```sql
-- +goose Up
CREATE TABLE post (
    id int NOT NULL,
    title text,
    body text,
    PRIMARY KEY(id)
);

-- +goose Down
DROP TABLE post;
```

Notice the annotations in the comments. Any statements following `-- +goose Up` will be executed as part of a forward migration, and any statements following `-- +goose Down` will be executed as part of a rollback.

By default, all migrations are run within a transaction. Some statements like `CREATE DATABASE`, however, cannot be run within a transaction. You may optionally add `-- +goose NO TRANSACTION` to the top of your migration 
file in order to skip transactions within that specific migration file. Both Up and Down migrations within this file will be run without transactions.

By default, SQL statements are delimited by semicolons - in fact, query statements must end with a semicolon to be properly recognized by goose.

More complex statements (PL/pgSQL) that have semicolons within them must be annotated with `-- +goose StatementBegin` and `-- +goose StatementEnd` to be properly recognized. For example:

```sql
-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION histories_partition_creation( DATE, DATE )
returns void AS $$
DECLARE
  create_query text;
BEGIN
  FOR create_query IN SELECT
      'CREATE TABLE IF NOT EXISTS histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || ' ( CHECK( created_at >= timestamp '''
      || TO_CHAR( d, 'YYYY-MM-DD 00:00:00' )
      || ''' AND created_at < timestamp '''
      || TO_CHAR( d + INTERVAL '1 month', 'YYYY-MM-DD 00:00:00' )
      || ''' ) ) inherits ( histories );'
    FROM generate_series( $1, $2, '1 month' ) AS d
  LOOP
    EXECUTE create_query;
  END LOOP;  -- LOOP END
END;         -- FUNCTION END
$$
language plpgsql;
-- +goose StatementEnd
```

## Go Migrations

1. Create your own goose binary, see [example](./examples/go-migrations)
2. Import `github.com/pressly/goose`
3. Register your migration functions
4. Run goose command, ie. `goose.Up(db *sql.DB, dir string)`

A [sample Go migration 00002_users_add_email.go file](./example/migrations-go/00002_rename_root.go) looks like:

```go
package migrations

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up, Down)
}

func Up(tx *sql.Tx) error {
	_, err := tx.Exec("UPDATE users SET username='admin' WHERE username='root';")
	if err != nil {
		return err
	}
	return nil
}

func Down(tx *sql.Tx) error {
	_, err := tx.Exec("UPDATE users SET username='root' WHERE username='admin';")
	if err != nil {
		return err
	}
	return nil
}
```

## License

Licensed under [MIT License](./LICENSE)

[GoDoc]: https://godoc.org/github.com/pressly/goose
[GoDoc Widget]: https://godoc.org/github.com/pressly/goose?status.svg
[Travis]: https://travis-ci.org/pressly/goose
[Travis Widget]: https://travis-ci.org/pressly/goose.svg?branch=master
