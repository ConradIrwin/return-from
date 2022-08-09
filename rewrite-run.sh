#!/usr/bin/env bash

set -e

go build rewrite.go
./rewrite "$1" > "$1".rewritten.go
go run "$1".rewritten.go