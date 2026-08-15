# Getting started with Redis 

This tutorial assumes that the latest version of dbctl is
[installed](../overview/install.md) and ready to use.

To start a test redis container run:

```shell
dbctl start rs
```

Output:
```shell
2023/09/24 22:36:20 Starting redis version 7.0.4 on port 16379 ...
2023/09/24 22:36:26 INFO: Wait for database to boot up
2023/09/24 22:36:26 Database uri is: "redis://localhost:16379/0"
```

By default `dbctl` is using `16379` port for redis. you can change it by passing the `-p` and a port number:

For example to use port `65474`:

```shell
dbctl start rs -p 65474
```

Passing `-p 0` lets dbctl pick a free port, so several projects can run their tests at the
same time. The port it settled on is printed in the connection uri, and the testing clients
find it on their own.

## Fixtures

`--fixtures` loads data into the database before your tests run. Files are applied in the
order of their names, and subdirectories are walked:

```shell
dbctl start rs -f ./fixtures
```

Files ending in `.lua` are evaluated as scripts, every other fixture file (`.redis`, `.txt`)
holds one command per line in redis-cli syntax. Blank lines and lines starting with `#` are
ignored, and values containing spaces can be quoted:

```
# fixtures/001_seed.redis
SET greeting "hello world"
HSET user:1 name "Ada Lovelace" born 1815
RPUSH list a b c
```

```lua
-- fixtures/002_more.lua
redis.call("SET", "from_lua", "yes")
```

The testing clients send their fixtures with every request, so each test gets its own
database index holding its own copy of the data.

To make sure start and stop commands are not effecting other instances of dbctl, you can pass a label to dbctl.
for more information please check [labels](../reference/labels.md) section.


