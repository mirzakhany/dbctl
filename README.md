# DbCtl

## Overview
DbCtl is a cli tools to create run databases for test and education purpose. 
it can run migration and setup fixtures.

## Install
To install `dbctl` from it source:

```shell
go install github.com/mirzakhany/dbctl@latest
```

## How to use
to run a postgres instance you can run:

```shell
dbctl start pg
```
this command with start a postgis instance using docker engine on port `15432`, with  `postgres` as default username,password and db name.
to get more detail run
```shell
dbctl start pg --help
```
```shell
Usage:
  dbctl start pg [flags]

Flags:
  -f, --fixtures string     Path to fixture files, its can be a file or directory.files in directory will be sorted by name before applying.
  -h, --help                help for pg
  -m, --migrations string   Path to migration files, will be applied if provided
  -n, --name string         Database name (default "postgres")
      --pass string         Database password (default "postgres")
  -p, --port uint32         postgres default port (default 15432)
  -u, --user string         Database username (default "postgres")
  -v, --version string      Database version, default for native 14.3.0 and 14.3.2 for docker engine
```

an example to run migrations and apply fixtures with be like this:

```shell
dbctl start pg -m ./migrations -f ./testdata
```
Out put:

```shell
2022/09/13 21:28:02 Starting postgres version 13-3.1 on port 15432 ...
2022/09/13 21:28:04 Wait for database to boot up
2022/09/13 21:28:07 Postgres is up and running
2022/09/13 21:28:07 Applying migrations ...
2022/09/13 21:28:07 Applying fixtures ...
2022/09/13 21:28:07 Database uri is: "postgres://postgres:postgres@localhost:65432/postgres?sslmode=disable"
```
