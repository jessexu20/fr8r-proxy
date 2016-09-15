#!/bin/bash
set -v

export GOPATH="$(pwd):$GOPATH"

mkdir -p bin/

go build -o bin/api-proxy src/api-proxy/api-proxy.go

ls -l bin
