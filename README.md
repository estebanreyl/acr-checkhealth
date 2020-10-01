![Go](https://github.com/aviral26/acr-checkhealth/workflows/Go/badge.svg?branch=main)
![Docker Image CI](https://github.com/aviral26/acr-checkhealth/workflows/Docker%20Image%20CI/badge.svg)

# Azure Container Registry - Check Health
This tool can be used to check various [ACR](https://aka.ms/acr) APIs to evaluate the health of your registry endpoints.

## Build
Use the `Makefile` to build locally:
```shell
make
```
Alternatively, build a docker image:
```shell
docker build -t acr .
```
To use the docker image, pass command arguments directly to `docker run acr`.

## Usage

```shell
aviral@Azure:~$ acr
NAME:
   acr - ACR Check Health - evaluate the health of a registry

USAGE:
   acr [global options] command [command options] [arguments...]

VERSION:
   0.1.0

AUTHOR:
   Aviral Takkar

COMMANDS:
   ping          ping registry endpoints
   check-health  check health of registry endpoints
   help, h       Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --trace        print trace logs (default: false)
   --help, -h     show help (default: false)
   --version, -v  print the version (default: false)
```

### `--trace`

Use this global option to print detailed HTTP requests.

> **Warning:** this will print secrets

## Examples
The following examples use admin credentials.

### Ping Registry

This will ping the ACR metadata endpoints with and without authentication and the ACR data endpoint without authentication.

```shell
aviral@Azure:~$ acr ping -u avtakkareus2euap -p *** -d avtakkareus2euap.eastus2euap.data.azurecr.io avtakkareus2euap.azurecr.io
{"level":"info","time":"2020-10-01T09:46:30Z","message":"DNS:  avtakkareus2euap.azurecr.io -> r0927cnre-2-az.eastus2euap.cloudapp.azure.com. -> 20.39.15.131"}
{"level":"info","time":"2020-10-01T09:46:31Z","message":"DNS:  avtakkareus2euap.eastus2euap.data.azurecr.io -> d0929cnre.eastus2euap.cloudapp.azure.com. -> 40.89.120.6"}
{"level":"info","time":"2020-10-01T09:46:31Z","message":"pinging frontend"}
{"level":"info","time":"2020-10-01T09:46:32Z","message":"pinging data proxy"}
{"level":"info","time":"2020-10-01T09:46:32Z","message":"ping was successful"}
```

### Check Health

This will try to push and pull a small OCI image.

```shell
aviral@Azure:~$ acr check-health -u avtakkareus2euap -p *** -d avtakkareus2euap.eastus2euap.data.azurecr.io avtakkareus2euap.azurecr.io
{"level":"info","time":"2020-10-01T09:47:14Z","message":"DNS:  avtakkareus2euap.azurecr.io -> r0927cnre-2-az.eastus2euap.cloudapp.azure.com. -> 20.39.15.131"}
{"level":"info","time":"2020-10-01T09:47:14Z","message":"DNS:  avtakkareus2euap.eastus2euap.data.azurecr.io -> d0929cnre.eastus2euap.cloudapp.azure.com. -> 40.89.120.6"}
{"level":"info","time":"2020-10-01T09:47:14Z","message":"pinging frontend"}
{"level":"info","time":"2020-10-01T09:47:15Z","message":"pinging data proxy"}
{"level":"info","time":"2020-10-01T09:47:15Z","message":"ping was successful"}
{"level":"info","time":"2020-10-01T09:47:15Z","message":"checking OCI push"}
{"level":"info","time":"2020-10-01T09:47:18Z","message":"checking OCI pull"}
{"level":"info","time":"2020-10-01T09:47:19Z","message":"check-health was successful"}
```

