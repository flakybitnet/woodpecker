#!/bin/sh
set -eu

set -a
. .ci/lib.sh
set +a

echo "Building $APP_NAME-$APP_COMPONENT UI"

export NPM_CONFIG_REGISTRY="$JS_PROXY"

cd ./web
corepack enable
pnpm install --frozen-lockfile
retry 3 pnpm build

echo 'Done'
