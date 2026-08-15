# Golang sdk

With the DBCTL Golang SDK, you can create ephemeral databases for each unit test, ensuring that each test has its own fresh database.

To begin, you'll need DBCTL running in testing mode. In this example, we want to set up a PostgreSQL database.

Start dbctl with a Postgres db
```shell
dbctl testing -- pg
```

Use sdk in test:

Install the sdk:
```shell
go get github.com/mirzakhany/dbctl/clients/dbctlgo
```

```golang
package foo

import (
	"database/sql"

	"testing"
	"github.com/mirzakhany/dbctl/clients/dbctlgo"
	// golang postgres driver
	_ "github.com/lib/pq"
)

func TestMustCreatePostgresDB(t *testing.T) {
	uri := dbctlgo.MustCreatePostgresDB(t)
	if uri == "" {
		t.Fatal("url is empty")
	}

	t.Log("uri:", uri)

	conn, err := sql.Open("postgres", uri)
	if err != nil {
		t.Fatal(err)
	}

	if err := conn.Ping(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = conn.Close()
	}()

	// create a table 
	res, err := conn.Exec("create table foo(id int, name varchar(20));")
	if err != nil {
		t.Fatal(err)
	}

    ...
}

```

We can also apply migrations before starting the test. 
```golang
func TestMustCreatePostgresDB(t *testing.T) {
	uri := dbctlgo.MustCreatePostgresDB(t, dbctlgo.WithMigrations("./test_sql/schema"), dbctlgo.WithFixtures("./test_sql/fixtures"))
	if uri == "" {
		t.Fatal("url is empty")
	}

	t.Log("uri:", uri)

    ...
}

```

Each time we call `MustCreatePostgresDB` it will create a new database, apply migrations and test file and will return connection url.

To have redis database, we can do the same by starting the redis testing db:
```shell
dbclt testing -- rs
```

And use `MustCreateRedisDB` to create redis databases.

```golang
package foo

import (
	"database/sql"

	"testing"
	"github.com/mirzakhany/dbctl/clients/dbctlgo"
	"github.com/gomodule/redigo/redis"
)

func TestRedis(t *testing.T) {
	uri := dbctlgo.MustCreateRedisDB(t)
	if uri == "" {
		t.Fatal("url is empty")
	}

	t.Log("uri:", uri)
}
```


## Configuration

The client reads `DBCTL_HOST` and `DBCTL_PORT`, so pointing a suite at a particular dbctl
server needs no change to the test code:

```shell
DBCTL_PORT=1989 go test ./...
```

Options override the environment for a single call:

```golang
uri := dbctlgo.MustCreatePostgresDB(t,
    dbctlgo.WithHost("localhost", 1989),
    dbctlgo.WithMigrations("./migrations"),
)
```

You do not have to tell the client which port the database itself runs on: the server looks
the running instance up. `WithInstance` is only needed when the instance was started with
credentials of your own.

Migrations may live in subdirectories, they are applied in the order of their paths, and
`*.down.sql` files are skipped.
