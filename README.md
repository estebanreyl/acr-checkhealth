![Go](https://github.com/aviral26/acr-checkhealth/workflows/Go/badge.svg?branch=main)
![Docker Image CI](https://github.com/aviral26/acr-checkhealth/workflows/Docker%20Image%20CI/badge.svg)

# [Azure Container Registry](https://aka.ms/acr) - Check Health
This tool can be used to check various ACR APIs to evaluate the health of your registry endpoints.

## Usage

```powershell
PS D:\acr-checkhealth> ./acr
NAME:
   acr - ACR Check Health - evaluate the health of a registry

USAGE:
   acr.exe [global options] command [command options] [arguments...]

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

### Ping Registry

```powershell
PS D:\acr-checkhealth> ./acr ping -u avtakkareus2euap -p <admin-access-key> -d avtakkareus2euap.eastus2euap.data.azurecr.io avtakkareus2euap.azurecr.io
{"level":"info","time":"2020-09-28T05:53:32-07:00","message":"DNS:  avtakkareus2euap.azurecr.io -> avtakkareus2euap.privatelink.azurecr.io. -> e43095e046a7461db1751272be587c98.trafficmanager.net. -> eus2euap-2-az.fe.azcr.io. -> eus2euap-2-acr-az-reg.trafficmanager.net. -> r0916cnre-2-az.eastus2euap.cloudapp.azure.com. -> 20.39.15.130"}
{"level":"info","time":"2020-09-28T05:53:33-07:00","message":"DNS:  avtakkareus2euap.eastus2euap.data.azurecr.io -> avtakkareus2euap.eastus2euap.data.privatelink.azurecr.io. -> eus2euap-0.data.azcr.io. -> eus2euap-acr-dp.trafficmanager.net. -> d0831cnre.eastus2euap.cloudapp.azure.com. -> 40.89.120.3"}
{"level":"info","time":"2020-09-28T05:53:33-07:00","message":"pinging frontend"}
{"level":"info","time":"2020-09-28T05:53:33-07:00","message":"pinging data proxy"}
{"level":"info","time":"2020-09-28T05:53:34-07:00","message":"success"}
```



### Check Health

```powershell
PS D:\acr-checkhealth> ./acr check-health -u avtakkareus2euap -p <admin-access-key> -d avtakkareus2euap.eastus2euap.data.azurecr.io avtakkareus2euap.azurecr.io
{"level":"info","time":"2020-09-28T05:57:28-07:00","message":"DNS:  avtakkareus2euap.azurecr.io -> avtakkareus2euap.privatelink.azurecr.io. -> e43095e046a7461db1751272be587c98.trafficmanager.net. -> eus2euap-2-az.fe.azcr.io. -> eus2euap-2-acr-az-reg.trafficmanager.net. -> r0916cnre-2-az.eastus2euap.cloudapp.azure.com. -> 20.39.15.130"}
{"level":"info","time":"2020-09-28T05:57:28-07:00","message":"DNS:  avtakkareus2euap.eastus2euap.data.azurecr.io -> avtakkareus2euap.eastus2euap.data.privatelink.azurecr.io. -> eus2euap-0.data.azcr.io. -> eus2euap-acr-dp.trafficmanager.net. -> d0831cnre.eastus2euap.cloudapp.azure.com. -> 40.89.120.3"}
{"level":"info","time":"2020-09-28T05:57:28-07:00","message":"pinging frontend"}
{"level":"info","time":"2020-09-28T05:57:29-07:00","message":"pinging data proxy"}
{"level":"info","time":"2020-09-28T05:57:30-07:00","message":"checking OCI push"}
{"level":"info","time":"2020-09-28T05:57:33-07:00","message":"checking OCI pull"}
{"level":"info","time":"2020-09-28T05:57:35-07:00","message":"success"}
```

