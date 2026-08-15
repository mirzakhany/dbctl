# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`dbctl` is a Go CLI that runs throwaway databases (Postgres, Redis, MongoDB) in Docker containers for testing: apply migrations/fixtures, print a connection URI, expose an HTTP API so test suites can create per-test databases, then clean everything up. Not for production use.

## Commands

```shell
make build          # go build -mod vendor -> ./dbctl
make install        # go install with version ldflags
make test           # go test ./...
make lint           # golangci-lint v1.53.3 via docker, config golangci.yaml
make vendor         # go mod vendor (deps are vendored; always re-run after changing go.mod)
make mod            # go get -u ./... + tidy + vendor
make build_docker   # mirzakhani/dbctl image
```

Single test: `go test ./internal/utils -run TestGetListHash -v`

Client integration tests (`make test_clients`) boot real containers via `make db_up`
(`go run main.go testing --label dbctl-client-test -- pg - rs`), then run the Go and Python
client suites against the live api-server, then `make db_down`. They need Docker running.
`clients/dbctlgo` and `clients/python` are separate modules/packages — build and test them from
their own directories.

## Architecture

Three layers, top to bottom:

**`cmd/`** — cobra command tree assembled in `main.go`. `cmd.GetRootCmd` defines the global
`--label` flag inherited by everything. Subcommand handlers only parse flags and translate them
into functional options; no logic lives here.

**`internal/database/`** — one package per engine (`postgres`, `redis`, `mongodb`), each with
`config.go` (functional `Option` type + a `supportedVersions` map of version string → docker image)
and the engine file implementing two interfaces from `database.go`:
- `Database` (`Start`/`Stop`/`WaitForStart`/`URI`) — lifecycle of the *container*.
- `Admin` (`CreateDB`/`RemoveDB`) — lifecycle of a *database inside* a running container. This is
  what the api-server drives.

Each package also exports a package-level `Instances(ctx, label)` that lists its running containers
by type — narrowed to the user's `--label` when one is given — used by `stop`/`ls`.

**`internal/container/`** — thin wrapper over the Docker SDK (`docker_rest.go`). All container
create/list/remove/exec goes through here. Containers are tagged with `dbctl_type` (engine name,
values in `database.Label*`) and `dbctl_custom` (the user's `--label`); every lookup is a label
query, which is how dbctl avoids touching unrelated containers.

### Adding a new engine

Touch all of: `internal/database/<engine>/{config.go,<engine>.go}` implementing `Database` +
`Admin` + `Instances`, a `database.Label<Engine>` constant, `cmd/start/<engine>.go` registered in
`cmd/start/start.go`, the type switch in `internal/apiserver/server.go` (`createDB`/`removeDB` and
the request-type validation list), and the arg matching in `cmd/stop.go`/`cmd/list.go`.

### Postgres template-database trick

`postgres.CreateDB` hashes the *content* of the migration files (`templateName` in `config.go`) and
uses that hash as a template database name. First call runs migrations into a fresh DB and
snapshots it as the template; later calls with identical migrations do
`create database X with template <hash>`, which is why per-test databases are fast.
`errDatabaseNotExists` is the sentinel that distinguishes "template missing, build it" from a real
failure. Fixtures are applied *after* the clone, so they are never baked into the template.

Hash the content, never the paths: the api server writes each request's uploads into a fresh
`os.MkdirTemp` directory, so a path-based hash never repeats and the template is never reused.

### API server and clients

`dbctl api-server` (default port 1988) exposes `POST /create` (multipart: migration and fixture
files uploaded, written to temp dirs, applied) and `DELETE /remove`. `dbctl testing -- pg - rs`
splits its args on the literal `-` separator, re-executes the root command as `start -d ...` for
each segment, then starts the api-server with `-t`, which runs the server *itself* in a container
(`apiserver.RunAPIServerContainer`).

The Go (`clients/dbctlgo`) and Python (`clients/python`) clients are HTTP clients for those two
endpoints; they select which local files to upload with a regex filter over the migrations/fixtures
directories. Both always send a multipart body, even with no files attached — the server reads its
scalar fields out of the multipart form.

**`DBCTL_INSIDE_DOCKER=true`** is set on the containerized api-server. When set, code connects to
databases via `host.docker.internal` but rewrites returned URIs back to `localhost` so the URI is
usable by the test process on the host. Any new engine must replicate both halves of this swap.

## Conventions

- Deps are vendored; builds use `-mod vendor`. Commit `vendor/` changes with `go.mod`/`go.sum`.
- Constructors are `New(options ...Option) (*T, error)` with `With*` option functions.
- Use `internal/logger` (`logger.Info/Debug/Error`), not `fmt.Println`, for user-facing output.
- Long-running commands get their context from `utils.ContextWithOsSignal()` for Ctrl-C cleanup.
- Interface conformance is asserted at package level: `var _ database.Database = (*Postgres)(nil)`.
- Docs live in `docs/` (Sphinx/readthedocs, published at dbctl.readthedocs.io); user-visible flag or
  command changes belong in `docs/reference/cli.md`.
- Releases are tag-driven (`v*` → GoReleaser workflow); `main.version` is injected via ldflags.
