# DBCTL

DBCTL runs throwaway databases in Docker so tests do not have to share one. It starts the
database, applies your migrations and sample data, prints a connection URL, and cleans up
after itself.

- Run a database with applied migration files and sample data
- Get the connection URL and use whatever tool you like to connect
- Launch a web UI for the database, where one is available
- Give every unit test its own fresh database through the Go and Python SDKs
- Clean the databases up when you are done

#### DBCTL is not intended for running databases in production. Its primary purpose is to simplify testing and practice with no hassle.

## Supported databases

| Database | Command   | Default port | Versions                                                       | Migrations | Fixtures            | UI            |
| -------- | --------- | ------------ | -------------------------------------------------------------- | ---------- | ------------------- | ------------- |
| Postgres | `pg`      | 15432        | 10.3.2, 11.2.5, 11.3.2, 12.3.2, 13-3.1, 13.3.2, 14.3.2 (default) | `.sql`     | `.sql`              | pgweb         |
| Redis    | `rs`      | 16379        | 7.0.4                                                           | –          | `.redis`/`.txt`/`.lua` | –          |
| MongoDB  | `mdb`     | 27017        | 6.0, 7.0 (default)                                              | `.js`/`.json` | `.js`/`.json`    | mongo-express |

Postgres images include PostGIS. Pass `--ui` to start the web UI where one exists.

## Install

```shell
go install github.com/mirzakhany/dbctl@latest
```

Or grab a prebuilt binary from the [releases](https://github.com/mirzakhany/dbctl/releases)
page, or run it from Docker:

```shell
docker run -lt --rm -v /var/run/docker.sock:/var/run/docker.sock mirzakhani/dbctl:latest /dbctl [options] [args]
```

Note: with a rootless docker installation the socket is at `$HOME/.docker/run/docker.sock`.

## Quick start

```shell
dbctl start pg -m ./migrations -f ./fixtures
```

```
INFO: Starting postgres version 14.3.2 ...
INFO: Postgres is up and running
INFO: Applying migrations ...
INFO: Database uri is: "postgres://postgres:postgres@localhost:15432/postgres?sslmode=disable"
```

Add `-d` to keep it running in the background, then list and stop what is running:

```shell
dbctl ls
dbctl stop pg          # or an instance id, a label, or all
```

Migrations and fixtures may be organised in subdirectories; files are applied in the order of
their paths, and `*.down.sql` files are skipped.

## A database per test

Start dbctl in testing mode, which brings up the databases and an API server the SDKs talk
to:

```shell
dbctl testing -- pg -m ./migrations - rs
```

Go ([docs](https://dbctl.readthedocs.io/en/latest/testing/golang.html)):

```golang
func TestUsers(t *testing.T) {
    uri := dbctlgo.MustCreatePostgresDB(t, dbctlgo.WithMigrations("./migrations"))
    // the database is removed when the test ends
}
```

Python ([docs](https://dbctl.readthedocs.io/en/latest/testing/python.html)):

```python
with dbctl.database(dbctl.DATABASE_POSTGRES, dbctl.Config().with_migrations("./migrations")) as uri:
    ...
```

The first database with a given set of migrations is snapshotted as a template, so the ones
after it are created from that snapshot instead of migrating again.

## Running several projects at once

`-p 0` takes any free port and `--api-port` moves the API server, so two projects can run
their tests at the same time. Clients do not need to know which port a database ended up on:
the API server looks the running instance up.

```shell
dbctl testing --label myproject --api-port 1989 -- pg -p 0
DBCTL_PORT=1989 go test ./...
```

Use `--label` to keep `stop` from touching instances that belong to another project:

```shell
dbctl stop myproject
```

## Reachability

Ports are published to `127.0.0.1`. The databases run with well known credentials, and
docker's rules sit in front of the host firewall, so publishing to every interface would hand
them to anyone on the same network. Pass `--listen <address>` when a database really has to
be reachable from elsewhere.

Check the [docs](https://dbctl.readthedocs.io) for the full reference.

## Todo

- [x] Setup and run postgres database
- [x] Setup and run redis
- [x] Setup and run MongoDB
- [x] A web base UI for Postgres and MongoDB
- [x] Support lua lang for redis fixtures
- [x] Javascript fixtures and migration scripts for MongoDB
- [ ] Javascript fixtures and migration scripts for the other databases
- [ ] Utilize golang templates to generate sample data
- [x] API server to let clients create databases
- [x] Golang client
- [x] Python client
- [ ] JS client

## Contributing
We welcome any and all contributions! Here are some ways you can get started:
1. Report bugs: If you encounter any bugs, please let us know. Open up an issue and let us know the problem.
2. Contribute code: If you are a developer and want to contribute, follow the instructions below to get started!
3. Suggestions: If you don't want to code but have some awesome ideas, open up an issue explaining some updates or improvements you would like to see!
4. Documentation: If you see the need for some additional documentation, feel free to add some!

## Instructions
1. Fork this repository
2. Clone the forked repository
3. Add your contributions (code or documentation)
4. Commit and push
5. Wait for pull request to be merged


## Supporters

JetBrains generously granted me a year of their open-source support licenses to work on this project.

<img src="https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.png" width="100" alt="JetBrains Logo (Main) logo.">
