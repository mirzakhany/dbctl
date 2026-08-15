# Python sdk

With the DBCTL Python SDK you can create an ephemeral database per test, so no test has to
clean up after another one.

Start dbctl in testing mode first. In this example we want a PostgreSQL database:

```shell
dbctl testing -- pg
```

Install the sdk:

```shell
pip install dbctl
```

## Creating a database

```python
import unittest
import psycopg2
from dbctl import dbctl


class TestUsers(unittest.TestCase):
    def test_users(self):
        uri = dbctl.must_create_postgres(
            dbctl.Config().with_migrations("./migrations").with_fixtures("./fixtures"))
        self.addCleanup(dbctl.remove_database, dbctl.DATABASE_POSTGRES, uri)

        conn = psycopg2.connect(uri)
        ...
```

`must_create_postgres`, `must_create_redis` and `must_create_mongodb` all return the
connection uri of a freshly created database. Removing it is up to you, either with
`addCleanup` as above or with the `database` context manager:

```python
from dbctl import dbctl

with dbctl.database(dbctl.DATABASE_POSTGRES, dbctl.Config().with_migrations("./migrations")) as uri:
    ...  # the database is removed when the block ends
```

## Configuration

`Config` accepts the same settings either as keyword arguments or through chained calls:

```python
config = dbctl.Config(
    migrations="./migrations",
    migrations_file_regex=r"^.*\.up\.sql$",
    fixtures="./fixtures",
    instance_port=15432,
    host_port=1988,
)

# or

config = (dbctl.Config()
          .with_migrations("./migrations", r"^.*\.up\.sql$")
          .with_fixtures("./fixtures")
          .with_instance("postgres", "postgres", "postgres", 15432)
          .with_host("localhost", 1988))
```

| setting | default | description |
| --- | --- | --- |
| `migrations` | *(none)* | directory holding the migration files, uploaded with the request |
| `migrations_file_regex` | *(none)* | only upload the migration files matching this pattern |
| `fixtures` | *(none)* | directory holding the fixture files |
| `with_default_migrations` | `False` | reuse the migrations the instance was started with instead of uploading them |
| `instance_user` / `instance_pass` / `instance_db_name` | *(per database type)* | credentials of the instance dbctl started, only needed when it was started with custom ones |
| `instance_port` | *(per database type)* | port the instance listens on, must match the `-p` used at start |
| `host_address` / `host_port` | `$DBCTL_HOST` / `$DBCTL_PORT`, else `localhost` / `1988` | address of the dbctl api server |

Migrations and fixtures may be organised in subdirectories; they are applied in the order of
their paths.

A migrations or fixtures path that does not exist raises `ErrDBCtl` rather than quietly
creating an empty database. Requests that fail server side raise `ErrDBCtl` with the
message returned by dbctl.

## Reusing the migrations of the instance

Uploading the same migrations with every test is unnecessary when dbctl already applied
them at start up:

```shell
dbctl testing -- pg -m ./migrations
```

```python
uri = dbctl.must_create_postgres(dbctl.Config().use_default_migrations())
```
