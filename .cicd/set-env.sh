#!/bin/sh
set -eu

echo Setting up environment

APP_NAME='woodpecker'
printf 'APP_NAME=%s\n' "$APP_NAME" >> .cicd/env

# from WP config
printf 'APP_COMPONENT=%s\n' "$APP_COMPONENT" >> .cicd/env

CI_COMMIT_SHA_SHORT=$(echo "$CI_COMMIT_SHA" | head -c 8)
if [ ! -z ${APP_TAG-} ]; then
    CI_COMMIT_TAG="$APP_TAG"
fi
APP_VERSION="${CI_COMMIT_TAG:-$CI_COMMIT_SHA_SHORT}"
printf 'APP_VERSION=%s\n' "$APP_VERSION" >> .cicd/env

printf 'HARBOR_REGISTRY=%s\n' 'harbor.flakybit.net' >> .cicd/env
printf 'HARBOR_PROJECT=%s\n' "$APP_NAME" >> .cicd/env
printf 'HARBOR_REPOSITORY=%s\n' "$APP_COMPONENT" >> .cicd/env

cat .cicd/env

echo Done
