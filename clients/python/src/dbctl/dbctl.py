from options import Config
import os
import requests
import json

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

    def __dict__(self):
        return {
            "uri": self.uri
        }
    
class RemoveDatabaseRequest(dict):
    db_type: str
    uri: str

    def __dict__(self):
        return {
            "type": self.db_type,
            "uri": self.uri
        }    



def must_create_database(config: Config):
    if config.database_type == DATABASE_POSTGRES:
        create_database(config, DATABASE_POSTGRES)
    elif config.database_type == DATABASE_REDIS:
        create_database(config, DATABASE_REDIS)
    else:
        raise ErrInvalideDatabaseType(f"Invalid database type: {config.database_type}")

def create_database(config: Config, db_type: str ):
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

    http_do_create_database(req, config.get_host_url())


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
        files.append(("migrations", open(file, "rb")))

    for file in fixtures_files:
        files.append(("fixtures", open(file, "rb")))

    req = requests.post(url, data=kv, files=files)    
    if req.status_code != 200:
        raise Exception(f"Error creating database: {req.text}")
    
    res = CreateDatabaseResponse()
    res.uri = req.json()["uri"]

    return res

def http_do_remove_database(req:RemoveDatabaseRequest, host_url: str):
    url = f"{host_url}/remove"
  
    req = requests.post(url, data=req.__dict__())
    if req.status_code != 204:
        raise Exception(f"Error removing database: {req.text}")

def get_files_list(path: str) -> list[str]:
    # retrun list of files in path
    return os.listdir(path)
