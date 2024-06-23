#!/bin/sh
set -eu

set -a
. .ci/lib.sh
set +a

echo "Setting authentication for $HARBOR_REGISTRY"
setRegistryAuth "$KANIKO_AUTH_FILE" "$HARBOR_REGISTRY" "$HARBOR_CREDS"

image="$APP_NAME/$APP_COMPONENT:$APP_VERSION"
dockerfile=".docker/Dockerfile-$APP_COMPONENT"

if [ "${IMAGE_DEBUG:-false}" = "true" ]; then
  image="$image-debug"
  dockerfile="$dockerfile-debug"
fi

echo "Building $image image"
executor -c ./ -f "$dockerfile" -d "$HARBOR_REGISTRY/$image"

echo 'Done'

