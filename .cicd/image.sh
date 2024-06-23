#!/bin/sh
set -eu

set -a
. .cicd/env
. .cicd/functions.sh
set +a

image="$HARBOR_PROJECT/$HARBOR_REPOSITORY:$APP_VERSION"
dockerfile=".docker/Dockerfile-$APP_COMPONENT"

if [ "${IMAGE_DEBUG:-false}" = "true" ]; then
  echo Debug image is set to build
  image="$image-debug"
  dockerfile="$dockerfile-debug"
fi

echo Building $image image

executor --context ./ \
    --dockerfile "$dockerfile" \
    --destination "$HARBOR_REGISTRY/$image"

echo Done

