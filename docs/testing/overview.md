# Testing with dbctl

DBCTL offers ephemeral databases for testing purposes, providing an advantage compared to running a database container in Docker Compose. With DBCTL, each test can have its own freshly prepared database.

## How it works
When you initiate DBCTL in testing mode, it will start the requested database containers and an API server that facilitates communication between the SDK and the database for creating databases.

To start DBCTL use `testing` command as below:

```shell
dbctl testing -- pg [options] - rs [options]
```

In this example, dbctl will set up both a PostgreSQL and a Redis database for testing purposes. You can also provide specific arguments for each of these databases, such as migrations, fixtures, or port numbers.

```shell
dbctl testing -- pg -m ./migrations - rs 
```

Output:
```shell
2023/09/26 19:43:22 INFO: Starting postgres version 13-3.1 on port 15432 ...
2023/09/26 19:43:22 INFO: Pulling docker image: "odidev/postgis:13-3.1-alpine", depends on your connection speed it might take upto minutes
2023/09/26 19:43:24 INFO: Wait for database to boot up
2023/09/26 19:43:26 INFO: Postgres is up and running
2023/09/26 19:43:26 INFO: Database uri is: "postgres://postgres:postgres@localhost:15432/postgres?sslmode=disable"
2023/09/26 19:43:26 INFO: Starting redis version 7.0.4 on port 16379 ...
2023/09/26 19:43:26 INFO: Pulling docker image: "redis:7.0.4-bullseye", depends on your connection speed it might take upto minutes
2023/09/26 19:43:27 INFO: Wait for database to boot up
2023/09/26 19:43:27 INFO: Database uri is: "redis://localhost:16379/0"
2023/09/26 19:43:27 INFO: Pulling docker image: "mirzakhani/dbctl:latest", depends on your connection speed it might take upto minutes
2023/09/26 19:43:29 INFO: Started apiserver on http://localhost:1988
```

You can see the containers running in background with `list` command:

```
dbclt ls
```

Output:
```shell
╭──────────────┬───────────────────────────────┬───────────╮
│ ID           │ Name                          │ Type      │
├──────────────┼───────────────────────────────┼───────────┤
│ a169b13b7456 │ /dbctl_apiserver_1695750207_2 │ apiserver │
│ 364dda9e71d6 │ /dbctl_rs_1695750206_5        │ redis     │
│ 0ca7542d688b │ /dbctl_pg_1695750202_1        │ postgres  │
╰──────────────┴───────────────────────────────┴───────────╯
```

The testing command runs the databases in detached mode, causing cli to exit when its are done. This means that you will need to manually stop them once your testing is complete.

```shell
dbctl stop all
```

Checkout the Golang SDK docs for how to use DBCTL to run unit tests in golang.
