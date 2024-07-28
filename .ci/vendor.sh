#!/bin/sh
set -eu

set -a
. .ci/lib.sh
set +a

echo Vendoring $APP_NAME

export GOPROXY="$GO_PROXY,https://proxy.golang.org,direct"
export GOPATH='/woodpecker/go'
export CGO_ENABLED=0

retry 2 go mod vendor

echo Done
