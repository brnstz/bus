#!/bin/bash

cleanup() {
    killall -9 busapi
}

error() {
    cleanup
    exit 1
}

cd `dirname $0` || error

psql -U $BUS_DB_USER -d $BUS_DB_NAME < ../schema/schema.sql || error
go install github.com/brnstz/bus/cmds/busapi || error
go install github.com/brnstz/bus/cmds/busloader || error

$GOPATH/bin/busloader || error
$GOPATH/bin/busapi & 

sleep 2

port=`echo $BUS_API_ADDR | cut -f2 -d:`
curl -i "http://localhost:$port/api/v1/stops?lat=40.729183&lon=-73.95154&&miles=0.5&filter=subway" | grep 'Greenpoint Av' || exit 1

cleanup
