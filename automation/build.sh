#!/bin/sh

# GOPATH_TEMP is a temporary directory where we build our code from scratch
GOPATH_TEMP=`mktemp -d`

# CODE_ROOT top level import path of our code
CODE_ROOT="github.com/brnstz/bus"

# BIN_DIR is the final location for binaries
BIN_DIR="./bin"

# Set the actual to GOPATH to the temporary directory
export GOPATH=$GOPATH_TEMP
export GOOS=linux
export GOARCH=amd64

# cleanup cleans our temporary directory
cleanup() {
    rm -rf $GOPATH_TEMP
}

# error runs cleanup before exiting
error() {
    cleanup
    exit 1
}

# Ensure we are running within this directory
cd `dirname $0` || error

# Ensure our binary directory exists
mkdir -p $BIN_DIR || error

# Copy code to temp directory
mkdir -p $GOPATH_TEMP/src/$CODE_ROOT || error
cp -R ../ $GOPATH_TEMP/src/$CODE_ROOT || error

# Get dependencies 
go get $CODE_ROOT/... || error

# Install our binaries
go build -o $BIN_DIR/busapi $CODE_ROOT/cmds/busapi || error
go build -o $BIN_DIR/busloader $CODE_ROOT/cmds/busloader || error

# Run web build
cd ../web || error
npm install || error
grunt || error

cleanup
