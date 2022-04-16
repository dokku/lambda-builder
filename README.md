# lambda-builder

A tool for building lamda functions into uploadable zip files via Docker.

## Building

```shell
# substitute the version number as desired
go build -ldflags "-X main.Version=0.1.0
```

## Usage

```
Usage: lambda-builder [--version] [--help] <command> [<args>]

Available commands are:
    build      Builds a lambda function
    version    Return the version of the binary
```
