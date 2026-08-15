from .options import Config
import os
import re
import requests

class ErrInvalideDatabaseType(Exception):
    pass

class ErrDBCtl(Exception):
    pass

DATABASE_POSTGRES = "postgres"
DATABASE_REDIS = "redis"
DATABASE_MONGODB = "mongodb"

DATABASE_TYPES = [DATABASE_POSTGRES, DATABASE_REDIS, DATABASE_MONGODB]


class CreateDatabaseRequest:
    db_type: str
    migrations: str
    migrations_file_regex: str
    fixtures: str

    with_default_migrations: bool

    instance_port: int
    instance_user: str
    instance_pass: str
    instance_name: str

    def __init__(self, db_type: str, migrations: str, fixtures: str, instance_port: int,
                 instance_user: str, instance_pass: str, instance_name: str,
                 migrations_file_regex: str = "", with_default_migrations: bool = False):
        self.db_type = db_type
        self.migrations = migrations
        self.migrations_file_regex = migrations_file_regex
        self.fixtures = fixtures
        self.with_default_migrations = with_default_migrations
        self.instance_port = instance_port
        self.instance_user = instance_user
        self.instance_pass = instance_pass
        self.instance_name = instance_name

    def form_fields(self) -> dict:
        return {
            "type": self.db_type,
            "instance_port": self.instance_port,
            "instance_user": self.instance_user,
            "instance_pass": self.instance_pass,
            "instance_name": self.instance_name,
            "with_default_migrations": str(self.with_default_migrations).lower(),
        }

class CreateDatabaseResponse:
    uri: str

    def __dict__(self):
        return {
            "uri": self.uri
        }

class RemoveDatabaseRequest:
    db_type: str
    uri: str

    def __init__(self, db_type: str, uri: str):
        self.db_type = db_type
        self.uri = uri

    def __dict__(self):
        return {
            "type": self.db_type,
            "uri": self.uri
        }


def must_create_postgres(config: Config = None) -> str:
    return must_create_database(DATABASE_POSTGRES, config)

def must_create_redis(config: Config = None) -> str:
    return must_create_database(DATABASE_REDIS, config)

def must_create_mongodb(config: Config = None) -> str:
    return must_create_database(DATABASE_MONGODB, config)

def must_create_database(database_type: str, config: Config = None) -> str:
    if database_type not in DATABASE_TYPES:
        raise ErrInvalideDatabaseType(f"Invalid database type: {database_type}")

    return create_database(config if config is not None else Config(), database_type)

def remove_database(database_type: str, uri: str, config: Config = None):
    http_do_remove_database(
        RemoveDatabaseRequest(
            db_type=database_type,
            uri=uri
        ),
        (config if config is not None else Config()).get_host_url()
    )


class database:
    """Context manager creating a database and removing it on exit.

    with dbctl.database(dbctl.DATABASE_POSTGRES, dbctl.Config().with_migrations("./migrations")) as uri:
        ...
    """

    def __init__(self, database_type: str, config: Config = None):
        self.database_type = database_type
        self.config = config if config is not None else Config()
        self.uri = None

    def __enter__(self) -> str:
        self.uri = must_create_database(self.database_type, self.config)
        return self.uri

    def __exit__(self, exc_type, exc_value, traceback):
        if self.uri is not None:
            remove_database(self.database_type, self.uri, self.config)
        return False


def create_database(config: Config, db_type: str) -> str:
    migrations_path = os.path.abspath(config.migrations) if config.migrations else ""
    fixtures_path = os.path.abspath(config.fixtures) if config.fixtures else ""

    req = CreateDatabaseRequest(
        db_type=db_type,
        migrations=migrations_path,
        migrations_file_regex=config.migrations_file_regex,
        fixtures=fixtures_path,
        with_default_migrations=config.with_default_migrations,
        instance_port=config.instance_port,
        instance_user=config.instance_user,
        instance_pass=config.instance_pass,
        instance_name=config.instance_db_name
    )

    res = http_do_create_database(req, config.get_host_url())
    return res.uri


def http_do_create_database(req: CreateDatabaseRequest, host_url: str) -> CreateDatabaseResponse:
    url = f"{host_url}/create"

    migration_files = get_files_list(req.migrations, req.migrations_file_regex)
    fixtures_files = get_files_list(req.fixtures, "")

    files = []
    opened = []
    try:
        for name in migration_files:
            handle = open(os.path.join(req.migrations, name), "rb")
            opened.append(handle)
            files.append(("migrations", (name, handle)))

        for name in fixtures_files:
            handle = open(os.path.join(req.fixtures, name), "rb")
            opened.append(handle)
            files.append(("fixtures", (name, handle)))

        # requests falls back to a urlencoded body when no file is attached, while
        # the server always expects a multipart one. this placeholder keeps the
        # encoding stable for requests without migrations or fixtures.
        if not files:
            files.append(("dbctl", ("dbctl", b"")))

        res = requests.post(url, data=req.form_fields(), files=files)
    finally:
        for handle in opened:
            handle.close()

    if res.status_code != 200:
        raise ErrDBCtl(f"Error creating database: {error_message(res)}")

    out = CreateDatabaseResponse()
    out.uri = res.json()["uri"]
    return out

def http_do_remove_database(req: RemoveDatabaseRequest, host_url: str):
    url = f"{host_url}/remove"

    res = requests.delete(url, json={"type": req.db_type, "uri": req.uri})
    if res.status_code != 204:
        raise ErrDBCtl(f"Error removing database: {error_message(res)}")

def error_message(res) -> str:
    try:
        return res.json().get("error", res.text)
    except ValueError:
        return res.text or f"server returned status {res.status_code}"

def get_files_list(path: str, regex_pattern: str = "") -> list[str]:
    # return the file names in path, optionally filtered by regex pattern
    if not path:
        return []

    if not os.path.isdir(path):
        raise ErrDBCtl(f"{path} is not an existing directory")

    files = [f for f in os.listdir(path) if os.path.isfile(os.path.join(path, f))]

    if regex_pattern:
        try:
            pattern = re.compile(regex_pattern)
        except re.error as e:
            raise ErrDBCtl(f"Invalid regex pattern: {e}") from e
        files = [f for f in files if pattern.search(f)]

    return sorted(files)
