// Environments
// RUN_COMPONENTS=server|agent|cli - execute workflow for components
// RUN_PHASES=build-image|publish-quay|publish-ghcr|publish-ecr - execute workflow phases
// CI_MANUAL_TAG=0.0.1 - application release version, gets priority over CI_COMMIT_TAG

local image = {
  debian: 'public.ecr.aws/docker/library/debian:bookworm-slim',
  golang: 'public.ecr.aws/docker/library/golang:1.23.11-bookworm',
  node: 'public.ecr.aws/docker/library/node:22.17.1-bookworm-slim',
  kaniko: 'gcr.io/kaniko-project/executor:v1.24.0-debug',
  skopeo: 'public.ecr.aws/flakybitnet/skopeo:1.19.0-fb1',
};

{
  matrix: {
    APP_COMPONENT: ['server', 'agent', 'cli'],
  },

  when: [
    {
      event: [ 'tag', 'manual'],
      evaluate: 'RUN_COMPONENTS == "" || APP_COMPONENT in split(RUN_COMPONENTS, ",")',
    },
    {
      event: ['push'],
      branch: { exclude: 'main' },
      evaluate: 'RUN_COMPONENTS == "" || APP_COMPONENT in split(RUN_COMPONENTS, ",")',
    },
  ],

  steps: [

    // prepare

    {
      name: 'set-env',
      image: image.debian,
      commands: ['.ci/set-env.sh'],
    },

    // build

    {
      name: 'vendor',
      when: {
        evaluate: 'RUN_PHASES == "" || "build-image" in split(RUN_PHASES, ",")',
      },
      image: image.golang,
      commands: ['.ci/vendor.sh'],
    },

    {
      name: 'build-ui',
      when: {
        evaluate: 'APP_COMPONENT == "server" && (RUN_PHASES == "" || "build-image" in split(RUN_PHASES, ","))',
      },
      image: image.node,
      commands: ['.ci/build-ui.sh'],
    },

    {
      name: 'build',
      when: {
        evaluate: 'RUN_PHASES == "" || "build-image" in split(RUN_PHASES, ",")',
      },
      image: image.golang,
      commands: ['.ci/build.sh'],
    },

    // image

    {
      name: 'image',
      when: {
        evaluate: 'RUN_PHASES == "" || "build-image" in split(RUN_PHASES, ",")',
      },
      image: image.kaniko,
      environment: {
        HARBOR_CREDS: { from_secret: 'fb_harbor_creds' },
      },
      commands: ['.ci/image.sh'],
    },

    {
      name: 'image-debug',
      when: {
        evaluate: 'RUN_PHASES == "" || "build-image" in split(RUN_PHASES, ",")',
      },
      image: image.kaniko,
      environment: {
        IMAGE_DEBUG: true,
        HARBOR_CREDS: { from_secret: 'fb_harbor_creds' },
      },
      commands: ['.ci/image.sh'],
    },

    // publish external

    {
      name: 'publish-quay',
      when: {
        evaluate: '(RUN_PHASES == "" || "publish-quay" in split(RUN_PHASES, ",")) && (CI_COMMIT_TAG != "" || CI_MANUAL_TAG != "")',
      },
      failure: 'ignore',
      image: image.skopeo,
      environment: {
        DEST_REGISTRY: 'quay.io',
        DEST_CREDS: { from_secret: 'fb_quay_creds' },
      },
      commands: ['.ci/publish-external.sh'],
    },

    {
      name: 'publish-ghcr',
      when: {
        evaluate: '(RUN_PHASES == "" || "publish-ghcr" in split(RUN_PHASES, ",")) && (CI_COMMIT_TAG != "" || CI_MANUAL_TAG != "")',
      },
      failure: 'ignore',
      image: image.skopeo,
      environment: {
        DEST_REGISTRY: 'ghcr.io',
        DEST_CREDS: { from_secret: 'fb_ghcr_creds' },
      },
      commands: ['.ci/publish-external.sh'],
    },

    {
      name: 'publish-ecr',
      when: {
        evaluate: '(RUN_PHASES == "" || "publish-ecr" in split(RUN_PHASES, ",")) && (CI_COMMIT_TAG != "" || CI_MANUAL_TAG != "")',
      },
      failure: 'ignore',
      environment: {
        DEST_REGISTRY: 'public.ecr.aws',
        AWS_ACCESS_KEY_ID: { from_secret: 'fb_ecr_key_id' },
        AWS_SECRET_ACCESS_KEY: { from_secret: 'fb_ecr_key' },
      },
      image: image.skopeo,
      commands: ['.ci/publish-external.sh'],
    },

  ],
}
