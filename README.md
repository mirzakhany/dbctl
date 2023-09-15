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

## Using docker

```shell
 docker run -lt --rm -v /var/run/docker.sock:/var/run/docker.sock  dbctl:latest /dbctl ls
```

### Start a postgres instance
```shell
dbctl start pg
```
this command will start a postgis instance using docker engine on port `15432`, with  `postgres` as default username,password and db name.

Flags:
```shell
Flags:
  -f, --fixtures string     Path to fixture files, its can be a file or directory.files in directory will be sorted by name before applying.
  -h, --help                help for postgres
  -m, --migrations string   Path to migration files, will be applied if provided
  -n, --name string         Database name (default "postgres")
      --pass string         Database password (default "postgres")
  -p, --port uint32         postgres default port (default 15432)
  -u, --user string         Database username (default "postgres")
  -v, --version string      Database version, default 14.3.2

Global Flags:
  -d, --detach   Detached mode: Run database in the background
      --ui       Run ui component if available for chosen database
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

### Start a redis instance
```shell
dbctl start rs
```
It will start a redis instance using docker engine on port `16379`, with no username and password.

Output:
```shell
2022/09/17 17:20:58 Starting redis version 7.0.4 on port 16379 ...
2022/09/17 17:21:01 Wait for database to boot up
2022/09/17 17:21:01 Database uri is: "redis://localhost:16379/0"
```

Flags:
```shell
      --db int           Redis db index
  -h, --help             help for redis
      --pass string      Database password
  -p, --port uint32      Redis default port (default 16379)
  -u, --user string      Database username
  -v, --version string   Database version, default 7.0.4 for docker engine
```
