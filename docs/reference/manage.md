# Manage running containers

You can use `dbctl ls`` to view running containers managed by dbctl. This is particularly useful when running dbctl in detached mode (with the `-d` flag), as it runs the container and exits.

To test it lets run a postgres database. 
```shell
dbctl start pg
```

In another terminal run flowing command:
```shell
dbctl ls
```

Example Output:
```shell
╭──────────────┬─────────────────────────┬──────────╮
│ ID           │ Name                    │ Type     │
├──────────────┼─────────────────────────┼──────────┤
│ 6511509bb314 │ /dbctl_pg_1695666553_11 │ postgres │
╰──────────────┴─────────────────────────┴──────────╯
```

To stop a container by its ID, use stop command:
```shell
dbctl stop 6511509bb314
```

Or to stop all containers managed by dbctl run:
```shell
dbctl stop all
```

You can also stop containers by their database type. for example flowing command will stop and redis container that managed by dbctl:
```shell
dbclt stop rs
```

Its possible to send multipe types at the same time as well: stop all postgres and redis databases.
```shell
dbclt stop rs pg
```
