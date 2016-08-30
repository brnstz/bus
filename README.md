# bus

[![Build Status](https://travis-ci.org/brnstz/bus.svg?branch=master)](https://travis-ci.org/brnstz/bus?branch=master)

*Beta version:* https://token.live

## Dependencies

* Go 1.6+ 
* PostgreSQL 9.3+ with PostGIS
* Flyway
* Redis
* NPM
* Grunt
* JQuery
* Bootstrap
* Leaflet

## Target platform

* Ubuntu 14 LTS

## Supported agencies 

| Agency                 | Live departures  |
|------------------------|------------------|
| MTA NYC Transit        | Yes,            


## Binaries

The full system consists of two binaries. Each binary can be configured
using environment variables and typically are run as daemons. They are both 
located under the `cmds/` directory.

## Shared Database Config

Since both binaries connect to the database, they share the following
config variables:

| Name               | Description                 | Default value    |
|--------------------|-----------------------------|------------------|
| `BUS_DB_ADDR`      | `host:port` of postgres     | `localhost:5432` |
| `BUS_DB_USER`      | The username to use         | `postgres`       |
| `BUS_DB_PASSWORD`  | The password to use         | empty            |
| `BUS_DB_NAME`      | The database name to use    | `postgres`       |

## `busapi`

`busapi` is the queryable API. 

### Config

| Name                        | Description                            | Default value     |
|-----------------------------|----------------------------------------|-------------------|
| `BUS_API_ADDR`              | `host:port` to listen on               | `0.0.0.0:8000`          |
| `BUS_REDIS_ADDR`            | `host:port` of redis                   | `localhost:6379`  |
| `BUS_MTA_BUSTIME_API_KEY`   |  API key for http://bustime.mta.info/  | *None*            |
| `BUS_MTA_DATAMINE_API_KEY`  |  API key for http://datamine.mta.info/ | *None*            |


## `busloader`

`busloader` downloads 
[GTFS](https://developers.google.com/transit/gtfs/) files and loads
them to the database. Typically, these files are updated periodically
from a well-known URL. The loader incorporates these updates to the 
database without losing old values.

### Config

| Name                        | Description                                                                              | Default value       |
|-----------------------------|------------------------------------------------------------------------------------------|---------------------|
| `BUS_TMP_DIR`               | Path to temporary directory                                                              |`os.TempDir()`       |
| `BUS_GTFS_URLS`             | Comma-separated path to GTFS zip URLs                                                   | *None*              |
| `BUS_ROUTE_FILTER`          | Comma-separated list of `route_id` values to filter on (i.e., *only* load these routes)  | *None (no filter)*  |
| `BUS_LOAD_FOREVER`          | Load forever (24 hour delay between loads) if `true`, exit after first load if `false`   |  `true`             |

### Example

```bash
# Load only the G and L train info and exit after initial load
export BUS_GTFS_URLS="http://web.mta.info/developers/data/nyct/subway/google_transit.zip"
export BUS_ROUTE_FILTER="G,L"
export BUS_LOAD_FOREVER="false"
busloader 
```

## Automation

In the `automation/` directory, there is a sample of how to fully deploy the
system. A full configuration for a deploy consists of an inventory file and a
`group_vars/` file. The included config is called `inventory_vagrant`. For 
security reasons (the API keys), the vars are encrypted in this repo. You can
create your own config and deploy it locally by doing the following:

```bash

# Create vagrant server
$ cd automation/vagrant
$ vagrant up
$ cd ../..

# Overwrite group vars with defaults
$ cd automation/group_vars
$ cp defaults.yml inventory_vagrant.yml

# Add your API keys
$ vim inventory_vagrant.yml
$ cd ../..

# Deploy the system
$ cd automation
$ ./build.sh && ./deploy.sh inventory_vagrant db_install.yml db_migrations.yml api.yml web.yml loader.yml

# If all goes well, system is available on http://localhost:8000
```
