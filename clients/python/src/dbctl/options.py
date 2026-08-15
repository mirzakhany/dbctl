DEFAULT_HOST_ADDRESS = "localhost"
DEFAULT_HOST_PORT = 1988

# The instance details default to empty: the server knows the defaults of each
# database type and fills in the missing ones. Sending the postgres defaults for
# every type would make dbctl try to log into redis as "postgres".
DEFAULT_INSTANCE_PORT = 0
DEFAULT_INSTANCE_USER = ""
DEFAULT_INSTANCE_PASS = ""
DEFAULT_INSTANCE_DB_NAME = ""


class Config:
    migrations: str
    migrations_file_regex: str
    fixtures: str

    with_default_migrations: bool

    instance_port: int
    instance_user: str
    instance_pass: str
    instance_db_name: str

    host_address: str
    host_port: int

    def __init__(self, **kwargs):
        # empty by default: a path is only sent when the caller asks for it, so a
        # missing directory is reported instead of silently producing an empty
        # database.
        self.migrations = kwargs.get("migrations", "")
        self.migrations_file_regex = kwargs.get("migrations_file_regex", "")
        self.fixtures = kwargs.get("fixtures", "")

        self.with_default_migrations = kwargs.get("with_default_migrations", False)

        self.instance_port = kwargs.get("instance_port", DEFAULT_INSTANCE_PORT)
        self.instance_user = kwargs.get("instance_user", DEFAULT_INSTANCE_USER)
        self.instance_pass = kwargs.get("instance_pass", DEFAULT_INSTANCE_PASS)
        self.instance_db_name = kwargs.get("instance_db_name", DEFAULT_INSTANCE_DB_NAME)

        self.host_address = kwargs.get("host_address", DEFAULT_HOST_ADDRESS)
        self.host_port = kwargs.get("host_port", DEFAULT_HOST_PORT)

    def use_default_migrations(self):
        """Reuse the migrations the instance was started with."""
        self.with_default_migrations = True
        return self

    def with_migrations(self, migrations: str, file_regex: str = ""):
        self.migrations = migrations
        self.migrations_file_regex = file_regex
        return self

    def with_fixtures(self, fixtures: str):
        self.fixtures = fixtures
        return self

    def with_instance(self, user: str, password: str, dbname: str, port: int):
        self.instance_user = user
        self.instance_pass = password
        self.instance_db_name = dbname
        self.instance_port = port
        return self

    def with_host(self, address: str, port: int):
        self.host_address = address
        self.host_port = port
        return self

    def get_host_url(self):
        return f"http://{self.host_address}:{self.host_port}"

    # kept for backwards compatibility with the singular names
    with_migration = with_migrations
    with_fixture = with_fixtures
