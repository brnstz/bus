#!/bin/bash

cd `dirname $0` || exit 1

psql -U $BUS_DB_USER -d $BUS_DB_NAME < ../schema/schema.sql || exit 1
go install github.com/brnstz/bus/cmds/busapi || exit 1
go install github.com/brnstz/bus/cmds/busloader || exit 1

$GOPATH/bin/busloader || exit 1

#port=`echo $BUS_API_ADDR | cut -f2 -d:`

#curl -i "http://localhost:$port/api/v1/stops?lat=40.729183&lon=-73.95154&&miles=0.5&filter=subway"
