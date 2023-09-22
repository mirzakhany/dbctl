class Config:
    migrations:str
    fixtures:str

    with_default_migration: bool

    instance_port: int
    instance_user: str
    instance_pass: str
    instance_db_name: str

    host_address: str
    host_port: int

    def __init__(self, **kwargs):
        self.migrations = kwargs.get("migrations", "migrations")
        self.fixtures = kwargs.get("fixtures", "fixtures")

        self.with_default_migration = kwargs.get("with_default_migration", False)

        self.instance_port = kwargs.get("instance_port", 15432)
        self.instance_user = kwargs.get("instance_user", "postgres")
        self.instance_pass = kwargs.get("instance_pass", "postgres")
        self.instance_db_name = kwargs.get("instance_db_name", "postgres")

        self.host_address = kwargs.get("host_address", "localhost")
        self.host_port = kwargs.get("host_port", 1988)

    def with_default_migration(self):
        self.with_default_migration = True
        return self
    
    def with_migration(self, migrations: str):
        self.migrations = migrations
        return self
    
    def with_fixture(self, fixtures: str):
        self.fixtures = fixtures
        return self
    
    def with_instance(self, user, name, dbname:str, port: int):
        self.instance_user = user
        self.instance_pass = name
        self.instance_db_name = dbname
        self.instance_port = port
        return self
    
    def with_host(self, address: str, port: int):
        self.host_address = address
        self.host_port = port
        return self
    
    def get_host_url(self):
        return f"http://{self.host_address}:{self.host_port}"