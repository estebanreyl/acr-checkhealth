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
   2d6eced

AUTHOR:
   Aviral Takkar

COMMANDS:
   ping             ping registry endpoints
   check-health     check health of registry endpoints
   check-referrers  check referrers data path (push, pull) based on https://github.com/opencontainers/artifacts/pull/29
   help, h          Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --trace        print trace logs with secrets (default: false)
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
10:19AM INF DNS:  avtakkareus2euap.azurecr.io -> r0927cnre-2-az.eastus2euap.cloudapp.azure.com. -> 20.39.15.131
10:19AM INF DNS:  avtakkareus2euap.eastus2euap.data.azurecr.io -> d0929cnre.eastus2euap.cloudapp.azure.com. -> 40.89.120.6
10:19AM INF pinging frontend
10:19AM INF pinging data proxy
10:19AM INF ping was successful
```

### Check Health

This will try to push and pull a small OCI image. Data integrity is verified - both the size and digest of the pushed data must match the pulled data for success.

```shell
aviral@Azure:~$ acr check-health -u avtakkareus2euap -p *** -d avtakkareus2euap.eastus2euap.data.azurecr.io avtakkareus2euap.azurecr.io
10:18AM INF DNS:  avtakkareus2euap.azurecr.io -> r0927cnre-2-az.eastus2euap.cloudapp.azure.com. -> 20.39.15.131
10:18AM INF DNS:  avtakkareus2euap.eastus2euap.data.azurecr.io -> d0929cnre.eastus2euap.cloudapp.azure.com. -> 40.89.120.6
10:18AM INF pinging frontend
10:18AM INF pinging data proxy
10:18AM INF ping was successful
10:18AM INF push OCI image acrcheckhealth1624530602:1624530602
10:18AM INF pull OCI image acrcheckhealth1624530602:1624530602
10:18AM INF check-health was successful
```

### Check Referrers

This will push a small OCI image, and an artifact that [references](https://github.com/opencontainers/artifacts/pull/29) it. The artifact is then pulled, followed by its subject.

```shell
aviral@Azure:~$ acr check-referrers -u avtakkareus2euap -p *** -d avtakkareus2euap.eastus2euap.data.azurecr.io avtakkareus2euap.azurecr.io
10:18AM INF DNS:  avtakkareus2euap.azurecr.io -> r0927cnre-2-az.eastus2euap.cloudapp.azure.com. -> 20.39.15.131
10:18AM INF DNS:  avtakkareus2euap.eastus2euap.data.azurecr.io -> d0929cnre.eastus2euap.cloudapp.azure.com. -> 40.89.120.6
3:32AM INF pinging frontend
3:32AM INF pinging data proxy
3:32AM INF ping was successful
3:32AM INF push OCI image acrcheckhealth1624530753:1624530753
3:32AM INF push OCI artifact acrcheckhealth1624530753:1624530753-art-1624530753
3:32AM INF pull OCI artifact acrcheckhealth1624530753:1624530753-art-1624530753
3:32AM INF subject for artifact acrcheckhealth1624530753:1624530753-art-1624530753 was pushed as acrcheckhealth1624530753:1624530753
3:32AM INF pull OCI image acrcheckhealth1624530753:1624530753
3:32AM INF check-referrers was successful
```
