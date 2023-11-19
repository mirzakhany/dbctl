from .options import Config
import os
import requests

class ErrInvalideDatabaseType(Exception):
    pass

DATABASE_POSTGRES = "postgres"
DATABASE_REDIS = "redis"


class CreateDatabaseRequest:
    db_type: str
    migrations: str
    fixtures: str

    instance_port: int
    instance_user: str
    instance_pass: str
    instance_name: str

    def __init__(self, db_type: str, migrations: str, fixtures: str, instance_port: int, instance_user: str, instance_pass: str, instance_name: str):
        self.db_type = db_type
        self.migrations = migrations
        self.fixtures = fixtures
        self.instance_port = instance_port
        self.instance_user = instance_user
        self.instance_pass = instance_pass
        self.instance_name = instance_name

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


def must_create_postgres(config: Config = Config()) -> str:
    return must_create_database(DATABASE_POSTGRES, config)

def must_create_redis(config: Config= Config()) -> str:
    return must_create_database(DATABASE_REDIS, config)

def must_create_database(database_type: str, config: Config= Config())-> str:
    if database_type == DATABASE_POSTGRES:
        return create_database(config, DATABASE_POSTGRES)
    elif database_type == DATABASE_REDIS:
        return create_database(config, DATABASE_REDIS)
    else:
        raise ErrInvalideDatabaseType(f"Invalid database type: {database_type}")

def remove_database(database_type: str, uri: str, config: Config = Config()):
    http_do_remove_database(
        RemoveDatabaseRequest(
            db_type=database_type,
            uri=uri
        ),
        config.get_host_url()    
    )


def create_database(config: Config, db_type: str) -> str:
    migrations_path: str
    fixtures_path: str

    if config.migrations != "":
        migrations_path = os.path.abspath(config.migrations)

    if config.fixtures != "":
        fixtures_path = os.path.abspath(config.fixtures)

    req = CreateDatabaseRequest(
        db_type=db_type,
        migrations=migrations_path,
        fixtures=fixtures_path,
        instance_port=config.instance_port,
        instance_user=config.instance_user,
        instance_pass=config.instance_pass,
        instance_name=config.instance_db_name
    )

    res = http_do_create_database(req, config.get_host_url())
    return res.uri


def http_do_create_database(req: CreateDatabaseRequest, host_url: str) -> CreateDatabaseResponse:
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

    files = []
    for file in migration_files:
        full_path = os.path.join(req.migrations, file)
        files.append(("migrations", open(full_path, "rb")))

    for file in fixtures_files:
        full_path = os.path.join(req.fixtures, file)
        files.append(("fixtures", open(full_path, "rb")))

    req = requests.post(url, data=kv, files=files)    
    if req.status_code != 200:
        raise Exception(f"Error creating database: {req.text}")
    
    res = CreateDatabaseResponse()
    res.uri = req.json()["uri"]

    for file in files:
        file[1].close()

    return res

def http_do_remove_database(req:RemoveDatabaseRequest, host_url: str):
    url = f"{host_url}/remove"
  
    res = requests.delete(url, json={"type": req.db_type, "uri": req.uri})
    if res.status_code != 204:
        raise Exception(f"Error removing database: {res.json()}")

def get_files_list(path: str) -> list[str]:
    # retrun list of files in path
    return os.listdir(path)
