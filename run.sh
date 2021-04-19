#!/bin/sh
set -e

echo "building binaries..."
mkdir -p ./bin
export GOOS=linux

go build -o ./bin/api ./cmd/api
go build -o ./bin/userservice ./cmd/userservice
go build -o ./bin/denormalizer ./cmd/denormalizer
go build -o ./bin/stats ./cmd/stats

echo "starting containers"
docker compose up
