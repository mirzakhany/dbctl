# DBCTL

## Description
DBCTL is a tool designed to make running a database in a Docker container easy and fast. It offers the following features:

- Run a database with applied migration files and sample data.
- Access the connection URL and use your preferred tools to connect to the database.
- Launch a user interface (UI) for managing the database, if available.
- Enable users to quickly run tests in a fresh database.
- Cleanup databases once you're finished using them!

#### DBCTL is not intended for running databases in production. Its primary purpose is to simplify testing and practice with no hassle.

## Install
To install `dbctl` from it source:

```shell
go install github.com/mirzakhany/dbctl@latest
```

### Using docker
```shell
 docker run -lt --rm -v /var/run/docker.sock:/var/run/docker.sock  dbctl:latest /dbctl [options] [args]
```

Note: in none root installation of docker `docker.sock` is in `$HOME/.docker/run/docker.sock`

You can also download a prebuilt binary from [releases](https://github.com/mirzakhany/dbctl/releases) page!


Please Check the [docs](https://dbctl.readthedocs.io) for usage.


### Todo

- [x] Setup and run postgres database
- [x] Setup and run redis
- [x] A web base UI for portgres
- [ ] Setup and run MongoDB
- [ ] Support lua lang for redis in fixtures and migration scripts
- [ ] Support for js in fixtures and migation scripts
- [ ] Utilize golang templates to generate sample data.
- [x] API server to let clients create databases
- [x] Golang client
- [ ] Python client (in progress)[PR](https://github.com/mirzakhany/dbctl/pull/17)
- [ ] JS client


## Contributing
We welcome any and all contributions! Here are some ways you can get started:
1. Report bugs: If you encounter any bugs, please let us know. Open up an issue and let us know the problem.
2. Contribute code: If you are a developer and want to contribute, follow the instructions below to get started!
3. Suggestions: If you don't want to code but have some awesome ideas, open up an issue explaining some updates or imporvements you would like to see!
4. Documentation: If you see the need for some additional documentation, feel free to add some!

## Instructions
1. Fork this repository
2. Clone the forked repository
3. Add your contributions (code or documentation)
4. Commit and push
5. Wait for pull request to be merged


## Supporters

JetBrains generously granted me a year of their open-source support licenses to work on this project.

<img src="https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.png" width="100" alt="JetBrains Logo (Main) logo.">