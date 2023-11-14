# Labeling running instances of dbctl

To make sure start and stop commands are not effecting other instances of dbctl, you can pass a label to dbctl.

```shell
dbctl start pg --label mydb
```

and now you can see the label in the list of running containers:

```shell
dbctl ls
```

Output:

```shell
╭──────────────┬────────────────────────┬──────────┬───────╮
│ ID           │ Name                   │ Type     │ Label │
├──────────────┼────────────────────────┼──────────┼───────┤
│ 24bcc1981511 │ /dbctl_pg_1699966841_3 │ postgres │ mydb  │
╰──────────────┴────────────────────────┴──────────┴───────╯
```

and you can pass the label to stop command as well:

```shell
dbctl stop mydb
```

Note that you can use any string as a label. dbctl will not validate it. and this label will be used for all 
database that are running together. 
for example if you have two databases (ex, a postgres and a redis) running with the same label, they will have the same label in the list of running containers.

