from options import Config
import os
import requests
from requests_toolbelt.multipart.encoder import MultipartEncoder

class ErrInvalideDatabaseType(Exception):
    pass

DATABASE_POSTGRES = "postgres"
DATABASE_REDIS = "redis"


class CreateDatabaseRequest(dict):
    db_type: str
    migrations: str
    fixtures: str

    instance_port: int
    instance_user: str
    instance_pass: str
    instance_name: str

    def __dict__(self):
        return {
            "type": self.db_type,
            "migrations": self.migrations,
            "fixtures": self.fixtures,
            "instance_port": self.instance_port,
            "instance_user": self.instance_user,
            "instance_pass": self.instance_pass,
            "instance_name": self.instance_name
        }

class CreateDatabaseResponse:
    uri: str


def must_create_database(config: Config):
    if config.database_type == DATABASE_POSTGRES:
        create_postgres_database(config)
    elif config.database_type == DATABASE_REDIS:
        create_redis_database(config)
    else:
        raise ErrInvalideDatabaseType(f"Invalid database type: {config.database_type}")

def create_postgres_database(config: Config):

    migrations_path: str
    fixtures_path: str

    if config.migrations != "":
        migrations_path = os.path.abspath(config.migrations)

    if config.fixtures != "":
        fixtures_path = os.path.abspath(config.fixtures)

    req = CreateDatabaseRequest(
        db_type=DATABASE_POSTGRES,
        migrations=migrations_path,
        fixtures=fixtures_path,
        instance_port=config.instance_port,
        instance_user=config.instance_user,
        instance_pass=config.instance_pass,
        instance_name=config.instance_db_name
    )

    http_do_create_database(req, config.get_host_url())

def create_redis_database(config: Config):
    pass



def http_do_create_database(req: CreateDatabaseRequest, host_url: str):
    url = f"{host_url}/create"

    migration_files = get_files_list(req.migrations)
    fixtures_files = get_files_list(req.fixtures)

    kv = {
        "type": req.db_type,
        "instance_port": req.instance_port,
        "instance_user": req.instance_user,
        "instance_pass": req.instance_pass,
        "instance_name": req.instance_name,
    }

    for i, f in enumerate(migration_files):
        kv[f"migrations[{i}]"] = f


def get_files_list(path: str):
    # retrun list of files in path
    pass
    