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


