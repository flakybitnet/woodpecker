# Woodpecker

Woodpecker is a simple yet powerful CI/CD engine with great extensibility.

## What is this?

This is a fork of [Woodpecker CI](https://github.com/woodpecker-ci/woodpecker) with the following changes:

1. It is based on top of the stable releases with back-porting.
2. It supports running the workload in Kubernetes [restricted](https://kubernetes.io/docs/concepts/security/pod-security-standards) environment,
   and Pod user namespaces also.
3. It supports secrets encryption.
4. It maintains self-cleanup tasks.
5. [Jsonnet](https://jsonnet.org/) support.
6. Other improvements.

## Release cadence

This fork maintains pace with upstream Woodpecker releases.
There is no predefined schedule, new versions are released as they are ready.

Our release versioning reflects the version of upstream Woodpecker that is being released. 
For example, the release `v2.6.0+fb1` maps to the `v2.6.0` Woodpecker release. 
We add a postfix in the form of `+fb<number>` to allow us to make additional releases using the same version of upstream Woodpecker. 
For example, if a some bug was fixed in the upstream `main`, we could release `v2.6.0+fb2`.

## Images

The OCI images are available at
* **GHCR**
  * [Server](https://github.com/flakybitnet/woodpecker/pkgs/container/woodpecker-server)
  * [Agent](https://github.com/flakybitnet/woodpecker/pkgs/container/woodpecker-agent)
  * [CLI](https://github.com/flakybitnet/woodpecker/pkgs/container/woodpecker-cli)

## Documentation

Please see [the official docs site](https://woodpecker-ci.org/docs/intro) for complete documentation.

### Restricted environment

You can run the workload (pipelines) in namespace with `restricted` [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/).
In order to achieve this, you should:

1. Label Woodpecker runtime namespace

   ```
   apiVersion: v1
   kind: Namespace
   metadata:
     name: wp-workload
     labels:
       pod-security.kubernetes.io/audit: restricted
       pod-security.kubernetes.io/enforce: restricted
       pod-security.kubernetes.io/warn: restricted
   ```

2. In Agent config set PSS profile and steps to run unprivileged

   ```
   WOODPECKER_BACKEND_K8S_PSS_PROFILE: restricted
   WOODPECKER_BACKEND_K8S_SECCTX_NONROOT: "true"
   WOODPECKER_BACKEND_K8S_SECCTX_USER: "1001"
   WOODPECKER_BACKEND_K8S_SECCTX_GROUP: "1001"
   WOODPECKER_BACKEND_K8S_SECCTX_FSGROUP: "1001"
   WOODPECKER_BACKEND_K8S_POD_USER_HOME: /woodpecker
   ```
   
   User and group are matter of your choice.

3. In Server config set fixed `plugin-git`

   ```
   WOODPECKER_DEFAULT_CLONE_IMAGE: harbor.flakybit.net/woodpecker/plugin-git:v2.5.0-fb1
   ```

---

Upstream issues:
 * [Add support for nonroot OCI images](https://github.com/woodpecker-ci/woodpecker/issues/1077)
 * [Add the ability to override default env variables for Kubernetes pipeline steps](https://github.com/woodpecker-ci/woodpecker/issues/3164)
 * [Cannot run pipeline on Kubernetes: CreateContainerError](https://github.com/woodpecker-ci/woodpecker/issues/2510)

### Secrets encryption

Based on upstream [AES secrets encryption](https://github.com/woodpecker-ci/woodpecker/pull/2300).

This is **_one-way_** operation. You cannot revert back storing secrets in database as plain text.

**_Make a backup!_**

In order to encrypt secrets set `WOODPECKER_SECRETS_ENCRYPTION_AES_KEY` with AES key.
You can generate the key using `openssl rand -base64 32`.

### Cleanup tasks

#### Stale agents

In order to clean stale Agents, in the Server config set `WOODPECKER_MAINTENANCE_CLEANUP_AGENTS_OLDER_THAN` with retention duration.

For example
```
WOODPECKER_MAINTENANCE_CLEANUP_AGENTS_OLDER_THAN=24h
```
will delete Agents last contacted more than 24 hour ago.

Upstream issue: [Agents cleaning](https://github.com/woodpecker-ci/woodpecker/issues/3023).

#### Pipeline logs

In order to clean old pipeline logs, in the Server config set `WOODPECKER_MAINTENANCE_CLEANUP_PIPELINE_LOGS_OLDER_THAN` with retention duration.

For example
```
WOODPECKER_MAINTENANCE_CLEANUP_PIPELINE_LOGS_OLDER_THAN=720h
```
will delete logs of pipelines created more than 30 days ago.

Upstream issue: [Delete old pipeline logs after X days or Y new runs](https://github.com/woodpecker-ci/woodpecker/issues/1068).

#### Stale K8s resources

If the Agent crashed while pipeline run, there will be abandoned Pod, PVC and maybe Service.
In order to clean stale resources, in the Agent config set `WOODPECKER_BACKEND_K8S_MAINTENANCE_CLEANUP_RESOURCES_OLDER_THAN` with retention duration.

For example
```
WOODPECKER_BACKEND_K8S_MAINTENANCE_CLEANUP_RESOURCES_OLDER_THAN=12h
```
will delete Kubernetes resources in the Agent's namespace created more than 12 hours ago.

The task runs once at the Agent startup.

### Jsonnet support

Based on upstream [Add Jsonnet support](https://github.com/woodpecker-ci/woodpecker/pull/1396).

[Jsonnet](https://jsonnet.org/) is a configuration language for app and tool developers.

You can now develop the pipelines using Jsonnet, for example
```jsonnet
{
  skip_clone: true,
  steps: {
    one: {
      image: 'alpine',
      commands: [
        std.join(' ', ['echo', 'Hello from', 'Jsonnet pipeline']),
        std.join(' ', ['echo', 'Hello from', self.image]),
      ],
    },
    two: {
      local ppStepNames = std.objectFields($.steps),
      image: 'alpine',
      commands: [
        'echo The number of steps is %d' % std.length(ppStepNames),
        'echo and they are: %(steps)s' % { steps: std.join(', ', ppStepNames) },
      ],
    },
  },
}
```

You can also import and use Woodpecker's environment variables:
```jsonnet
local env = import 'env.jsonnet';
{
  steps: {
    hello: {
      image: 'alpine',
      commands: [
        std.join(' ', ['echo', 'Hello', self.image, '!']),
        'echo Env vars are %s' % std.join(', ', std.objectFields(env)),
      ],
    },
  },
}
```

Upstream issue: [Support for Jsonnet](https://github.com/woodpecker-ci/woodpecker/discussions/3277)

## License

This fork of Woodpecker CI is distributed under [GNU Affero General Public License v3.0](LICENSE)
with the source files in this repository having a header indicating which license they are under and what copyrights apply.

Files under the `docs/` folder are licensed under Creative Commons Attribution-ShareAlike 4.0 International Public License.
