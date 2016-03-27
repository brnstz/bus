#!/bin/bash

cd `dirname $0` || error

psql -U $BUS_DB_USER < ../schema/schema.sql
go install github.com/brnstz/bus/cmds/busapi
$GOPATH/bin/busapi &
sleep 2
curl -i "http://localhost:$BUS_API_PORT/api/v1/stops?lat=40.729183&lon=-73.95154&&miles=0.5&filter=subway"
