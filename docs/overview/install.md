# Installing dbctl

dbctl is distributed as a single binary.

## Install from source

```shell
go install github.com/mirzakhany/dbctl@latest
```

## To run using docker

```shell
 docker run -lt --rm -v /var/run/docker.sock:/var/run/docker.sock  dbctl:latest /dbctl [options] [args]
```

Note: in none root installation of docker docker.sock is in `$HOME/.docker/run/docker.sock`

## Downloads

Get pre-built binaries for latest version are availabe from [releases](https://github.com/mirzakhany/dbctl/releases) pages
