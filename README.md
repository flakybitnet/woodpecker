# Woodpecker

Woodpecker is a simple yet powerful CI/CD engine with great extensibility.

## What is this?

This is a fork of [Woodpecker CI](https://github.com/woodpecker-ci/woodpecker) with the following changes:

1. It is based on top of the stable releases with back-porting.
2. It supports running the workload in Kubernetes [restricted](https://kubernetes.io/docs/concepts/security/pod-security-standards) environment.
3. It supports (one-way) secrets encryption.

## Release cadence

This fork maintains pace with upstream Woodpecker releases.
There is no predefined schedule, new versions are released as they are ready.

Our release versioning reflects the version of upstream Woodpecker that is being released. 
For example, the release `v2.6.0+fb1` maps to the `v2.6.0` Woodpecker release. 
We add a postfix in the form of `+fb<number>` to allow us to make additional releases using the same version of upstream Woodpecker. 
For example, if a some bug was fixed in the upstream `main`, we could release `v2.6.0+fb2`.

## Documentation
Please see [the official docs site](https://woodpecker-ci.org/docs/intro) for complete documentation.

## Images

The OCI images are available at
* **Quay** TBD
  * ~~[Server](https://quay.io/repository/flakybitnet/woodpecker-server)~~
  * ~~[Agent](https://quay.io/repository/flakybitnet/woodpecker-agent)~~
  * ~~[CLI](https://quay.io/repository/flakybitnet/woodpecker-cli)~~
* **GHCR** TBD
  * ~~[Server](https://github.com/flakybitnet/woodpecker/pkgs/container/woodpecker-server)~~
  * ~~[Agent](https://github.com/flakybitnet/woodpecker/pkgs/container/woodpecker-agent)~~
  * ~~[CLI](https://github.com/flakybitnet/woodpecker/pkgs/container/woodpecker-cli)~~
* **AWS ECR Public** TBD
  * ~~[Server](https://gallery.ecr.aws/flakybitnet/woodpecker/server)~~
  * ~~[Agent](https://gallery.ecr.aws/flakybitnet/woodpecker/agent)~~
  * ~~[CLI](https://gallery.ecr.aws/flakybitnet/woodpecker/cli)~~
* **FlakyBit's Harbor**
  * Server: `harbor.flakybit.net/woodpecker/server:<version>`
  * Agent: `harbor.flakybit.net/woodpecker/agent:<version>`
  * CLI: `harbor.flakybit.net/woodpecker/cli:<version>`

## License

Woodpecker is Apache 2.0 licensed with the source files in this repository having a header indicating which license they are under and what copyrights apply.

Files under the `docs/` folder are licensed under Creative Commons Attribution-ShareAlike 4.0 International Public License.
