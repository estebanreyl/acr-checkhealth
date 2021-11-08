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
docker build -t acr -f DOCKERFILE https://github.com/aviral26/acr-checkhealth.git#main
```

## Usage

```shell
aviral@Azure:~$ docker run acr
NAME:
   acr - ACR Check Health - evaluate the health of a registry

USAGE:
   acr [global options] command [command options] [arguments...]

AUTHOR:
   Aviral Takkar

COMMANDS:
   ping             ping registry endpoints
   check-health     check health of registry endpoints
   check-referrers  check referrers data path (push, pull) based on https://github.com/opencontainers/artifacts/pull/29
   help, h          Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --trace     print trace logs with secrets (default: false)
   --help, -h  show help (default: false)
```

### `--trace`

Use this global option to print detailed HTTP requests.

> **Warning:** this will print secrets

## Examples
The following examples use admin credentials.

### Ping Registry

This will ping the ACR metadata endpoints with and without authentication and the ACR data endpoint without authentication.

```shell
aviral@Azure:~$ docker run acr ping -u $user -p $pwd -d $dataendpoint $registry
10:41AM INF DNS:  avtakkareus2euapaz.azurecr.io -> r1029cnre-2-az.eastus2euap.cloudapp.azure.com. -> x.y.z.w
10:41AM INF DNS:  avtakkareus2euapaz.eastus2euap.data.azurecr.io -> d1029cnre-2-az.eastus2euap.cloudapp.azure.com. -> x.y.z.w
10:41AM INF pinging frontend
10:41AM INF pinging data proxy
10:41AM INF ping was successful
```

### Check Health

This will try to push and pull a small OCI image. Data integrity is verified - both the size and digest of the pushed data must match the pulled data for success.

```shell
aviral@Azure:~$ docker run acr check-health -u $user -p $pwd -d $dataendpoint $registry
10:42AM INF DNS:  avtakkareus2euapaz.azurecr.io -> r1029cnre-2-az.eastus2euap.cloudapp.azure.com. -> x.y.z.w
10:42AM INF DNS:  avtakkareus2euapaz.eastus2euap.data.azurecr.io -> d1029cnre-2-az.eastus2euap.cloudapp.azure.com. -> x.y.z.w
10:42AM INF pinging frontend
10:42AM INF pinging data proxy
10:42AM INF ping was successful
10:42AM INF push OCI image acrcheckhealth1636368134:1636368134
10:42AM INF sha256:6e5f4da7a1db602a6d7e911a8b885da4c78eccab8f18ce3c49d5cd41a8d44d77
10:42AM INF pull OCI image acrcheckhealth1636368134:1636368134
```

### Check Referrers

This will push a small OCI image, and an artifact that [references](https://github.com/opencontainers/artifacts/pull/29) it. The artifact is then discovered using the [/referrers API](https://gist.github.com/aviral26/ca4b0c1989fd978e74be75cbf3f3ea92), then pulled followed by its subject.

```shell
aviral@Azure:~$ docker run acr check-referrers -u $user -p $pwd --referrers 2 $registry
10:42AM INF DNS:  avtakkareus2euapaz.azurecr.io -> r1029cnre-2-az.eastus2euap.cloudapp.azure.com. -> x.y.z.w
10:42AM INF pinging frontend
10:42AM INF ping was successful
10:42AM INF push OCI image acrcheckhealth1636368170:1636368170
10:42AM INF sha256:1baff5e1d2aaf707a1629f7f095179ef8b50d29ddb03e9f444ffae009bcae816
10:42AM INF push ORAS artifact acrcheckhealth1636368170:art-1-1636368173
10:42AM INF sha256:d8ae624a47482a45f6a02e4839cd77911fe47baf8859198002e6a703bd1d522b
10:42AM INF push ORAS artifact acrcheckhealth1636368170:art-2-1636368174
10:42AM INF sha256:f0541156c9f1fb768430f97d0d33c771527afd73dc752647c4c8f356f0c514ea
10:42AM INF discover referrers for acrcheckhealth1636368170@sha256:1baff5e1d2aaf707a1629f7f095179ef8b50d29ddb03e9f444ffae009bcae816
10:42AM INF found 2 referrers
10:42AM INF sha256:d8ae624a47482a45f6a02e4839cd77911fe47baf8859198002e6a703bd1d522b
10:42AM INF sha256:f0541156c9f1fb768430f97d0d33c771527afd73dc752647c4c8f356f0c514ea
10:42AM INF pull referrer acrcheckhealth1636368170@sha256:d8ae624a47482a45f6a02e4839cd77911fe47baf8859198002e6a703bd1d522b
10:42AM INF pull referrer acrcheckhealth1636368170@sha256:f0541156c9f1fb768430f97d0d33c771527afd73dc752647c4c8f356f0c514ea
10:42AM INF subject is acrcheckhealth1636368170:1636368170
10:42AM INF pull OCI image acrcheckhealth1636368170:1636368170
10:42AM INF check-referrers was successful
```
