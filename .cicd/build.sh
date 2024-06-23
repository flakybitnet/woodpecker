#!/bin/sh
set -eu

set -a
. .cicd/env
. .cicd/functions.sh
set +a

echo Building $APP_NAME-$APP_COMPONENT

export GOPATH='/woodpecker/go'
export CGO_ENABLED=0

xldflags=""
xldflags="$xldflags -X go.woodpecker-ci.org/woodpecker/v2/version.Version=$APP_VERSION"

go build -v -ldflags "-s -w $xldflags" -o "dist/$APP_COMPONENT" "go.woodpecker-ci.org/woodpecker/v2/cmd/$APP_COMPONENT"

mkdir -p ./dist/etc
ls -lh ./dist

echo Done
