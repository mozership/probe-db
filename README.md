# probe-db

A [Probe](https://github.com/mozership/probe) action that runs SQL against MySQL, PostgreSQL and SQLite: it connects to the database a DSN names, runs one statement with the values given for its placeholders, and returns the rows it read or the count of rows it changed.

It is an [external action](https://probe.linyo.ws/guide/concepts/actions#external-actions), and takes the place of the `db` action that is built into Probe, with the same parameters and the same result. Probe downloads it the first time a workflow uses it, and runs the executable whose SHA-256 the `action.yml` at the pinned commit names. It needs Probe v1.41.0 or later, which lets a workflow give it the name `db`.

```yaml
name: Users
actions:
  db: github.com/mozership/probe-db@<commit SHA>
jobs:
- name: users
  defaults:
    db:
      dsn: "postgres://{{vars.user}}:{{vars.pass}}@localhost:5432/app?sslmode=disable"
  steps:
  - name: Add a user
    uses: db
    with:
      query: INSERT INTO users (name, email) VALUES ($1, $2)
      params: [Alice, alice@example.com]
    test: res.code == 0 && res.rows_affected == 1

  - name: Find the user
    uses: db
    with:
      query: SELECT id, name FROM users WHERE email = $1
      params: [alice@example.com]
    test: res.code == 0 && res.rows[0].name == "Alice"
```

The workflow gives the action a name once, under `actions`, and its steps use the name. Named `db`, it runs the steps a workflow wrote for the built-in `db` action as they are: adding the line under `actions` is all a workflow moving to probe-db changes. Probe only takes a full 40-character commit SHA there. The notes of each [release](https://github.com/mozership/probe-db/releases) start with the action and its commit to copy.

## Parameters

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `dsn` | String | Yes | | Where to connect, as a URL whose scheme picks the driver: `mysql://`, `postgres://` or `postgresql://`, and `file:` for SQLite |
| `query` | String | Yes | | The statement to run |
| `params` | List | No | | The values of the placeholders of `query`: `?` for MySQL and SQLite, `$1`, `$2` and on for PostgreSQL. They are sent apart from the statement, never written into it |
| `timeout` | Duration | No | `30s` | How long connecting and running the statement may take: a duration such as `"60s"`, or a number of seconds, above zero |

`driver` is taken too, for the workflows written for the built-in action, but the scheme of `dsn` decides the driver.

```yaml
# MySQL
dsn: "mysql://user:password@localhost:3306/app"
# PostgreSQL
dsn: "postgres://user:password@localhost:5432/app?sslmode=disable"
# SQLite: a file, relative to the working directory, or in memory
dsn: "file:./testdata/app.db"
dsn: "file::memory:"
```

The password in a DSN is hidden in the output and the reports of the run, as the other secrets of a workflow are.

## Result

| Field | Type | Description |
|---|---|---|
| `res.code` | Number | `0` when the statement ran, `1` when it failed |
| `res.rows_affected` | Number | The rows a write changed, or the rows a read returned |
| `res.rows` | List | The rows a read returned, each a map by column |
| `res.error` | String | Why the statement failed, as the database or the driver tells it |
| `status` | Number | `0` when the statement ran |
| `rt.duration`, `rt.sec` | String, Number | How long it took |

A statement whose first word is `SELECT`, `SHOW`, `DESCRIBE`, `DESC`, `EXPLAIN` or `WITH`, in any case, is a read and returns rows; any other is a write and returns the count of rows it changed.

A statement the database rejects, a connection that fails, or one that runs out of time, does not stop the step: `res.code` is `1` and `res.error` says why, so the test can check for the failure it expects.

```yaml
- name: The table is not there
  uses: db
  with:
    dsn: "file:./testdata/app.db"
    query: SELECT * FROM missing_table
  test: res.code == 1 && res.error contains "no such table"
```

## Under a guard

The action keeps to both kinds of guard, as `action.yml` declares with `guard: [read-only, allow-host]`.

- Under `--read-only`, it runs only one statement that reads, by its first word, and refuses any other before connecting, as well as a query that holds a semicolon anywhere but at its end. It runs that statement in a read-only transaction, or on a SQLite connection that only queries, so that the database itself refuses a write the statement hides, such as `WITH d AS (DELETE ...) SELECT ...`. It also refuses a DSN that would have the driver run a statement as the connection opens, before that transaction: for MySQL `multiStatements` and any system variable, and for SQLite any parameter but `mode`, `cache`, `immutable`, `_txlock` and `_time_format`.
- Under `--allow-host`, it refuses a DSN unless the run allows each server the driver may connect to, at the driver's port when none is named. For PostgreSQL they are the servers lib/pq resolves the DSN to, its parameters, a service file and `PGHOST`, `PGHOSTADDR`, `PGPORT` and the like included. A SQLite file names no host.

A refused step fails with the kind `refused`.

## Development

```bash
go test ./...
go build -o e2e/action/probe-db . && probe e2e/sqlite.yml
```

The workflows under `e2e` run the action through Probe against SQLite, PostgreSQL and MySQL, and again under `--read-only`; CI starts the servers. They name the action as it was just built `db`, as a workflow names a release.

## License

MIT. The licenses of the modules the executables are built from are published with each release, in `THIRD_PARTY_LICENSES`.
