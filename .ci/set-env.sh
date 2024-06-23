#!/bin/sh
set -eu

set -a
. .ci/lib.sh
set +a

echo Setting up environment

app_name='woodpecker'
printf 'APP_NAME=%s\n' "$app_name" >> "$CI_ENV_FILE"

# from WP config
printf 'APP_COMPONENT=%s\n' "$APP_COMPONENT" >> "$CI_ENV_FILE"

printf 'APP_VERSION=%s\n' "$(getAppVersion)" >> "$CI_ENV_FILE"
printf 'APP_RELEASE=%s\n' "$(getAppRelease)" >> "$CI_ENV_FILE"

printf 'HARBOR_REGISTRY=%s\n' 'harbor.flakybit.net' >> "$CI_ENV_FILE"
printf 'EXTERNAL_REGISTRY_NAMESPACE=%s\n' 'flakybitnet' >> "$CI_ENV_FILE"

printf 'KANIKO_AUTH_FILE=%s\n' '/kaniko/.docker/config.json' >> "$CI_ENV_FILE"
printf 'AWS_CREDS_FILE=%s\n' '.ci/aws-ecr-creds' >> "$CI_ENV_FILE"

printf 'GO_PROXY=%s\n' 'http://cache-go.proxy' >> "$CI_ENV_FILE"
printf 'JS_PROXY=%s\n' 'http://cache-js.proxy' >> "$CI_ENV_FILE"

cat "$CI_ENV_FILE"

echo Done
