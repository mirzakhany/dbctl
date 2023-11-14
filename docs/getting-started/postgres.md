# Getting started with PostgreSQL 

This tutorial assumes that the latest version of dbctl is
[installed](../overview/install.md) and ready to use.

To start a test postgres container run:

```shell
dbctl start pg
```

Output:
```shell
2023/09/24 21:24:10 INFO: Starting postgres version 13-3.1 on port 15432 ...
2023/09/24 21:24:12 INFO: Wait for database to boot up
2023/09/24 21:24:14 INFO: Postgres is up and running
2023/09/24 21:24:14 INFO: Database uri is: "postgres://postgres:postgres@localhost:15432/postgres?sslmode=disable"
```

By default `dbctl` is using `15432` port for postgres. you can change it by passing the `-p` and a port number:

For example to use port `65474`:

```shell
dbctl start pg -p 65474
```

You can also run the migrations by passing the directory which contains the migration files. please note that dbctl will sort files by name before applying them.
We recommend you to start migrations file with numbers to maintain desired order. 

```shell
dbctl start pg -m ./migrations
```

To add some test data to your newly created database you can use:

```shell
dbctl start pg -m ./migrations -f ./fixtures
```

If you need a web ui for managing you postgres database, dbctl provides a UI using [pgweb](https://github.com/sosedoff/pgweb) project. 


```shell
dbctl start pg --ui
```

Output:
```
2023/09/24 21:33:32 INFO: Starting postgres version 13-3.1 on port 15432 ...
2023/09/24 21:33:38 INFO: Wait for database to boot up
2023/09/24 21:33:40 INFO: Postgres is up and running
2023/09/24 21:33:40 INFO: Database uri is: "postgres://postgres:postgres@localhost:15432/postgres?sslmode=disable"
2023/09/24 21:33:40 INFO: Starting postgres ui using pgweb (https://github.com/sosedoff/pgweb)
2023/09/24 21:34:11 INFO: Database UI is running on: http://localhost:8081
```

By running any combination of above command, you get a running postgres database. by pressing `CTRL+C` dbctl will shutting down and destroy the database container.


#### Are your running multiple instances of dbctl?
To make sure start and stop commands are not effecting other instances of dbctl, you can pass a label to dbctl.
for more information please check [labels](../reference/labels.md) section.
