#!/bin/sh
set -eu

set -a
. .ci/lib.sh
set +a

echo "Building $APP_NAME-$APP_COMPONENT UI"

cd ./web
corepack enable
pnpm install --frozen-lockfile
retry 3 pnpm build

echo 'Done'
