#!/bin/sh
set -eu

set -a
. .cicd/env
. .cicd/functions.sh
set +a

echo Building $APP_NAME-$APP_COMPONENT UI

cd ./web
corepack enable
pnpm install --frozen-lockfile
retry 2 pnpm build

echo Done
