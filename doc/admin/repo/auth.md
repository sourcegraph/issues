# Repositories that need HTTP(S) or SSH authentication

If authentication is required to `git clone` a repository then you must provide credentials to the container.

First, ensure your **Site admin > Manage repositories** code host configuration is configured to use SSH. For example, by setting the `gitURLType` field to "ssh". Alternatively, you may use the "Generic Git host" code host type, which allows you to directly specify Git repository URLs for cloning.

Then, follow the directions below depending on your deployment type.

## For single-node deployments (`sourcegraph/server`)

### SSH authentication (config, keys, `known_hosts`)

The container consults its own file system (in the standard locations) for SSH configuration, private keys, and `known_hosts`. Upon container start, it copies all files from `/etc/sourcegraph/ssh` into its own `$HOME/.ssh` directory.

To provide SSH authentication configuration to the container, assuming you're using the default `--volume $HOME/.sourcegraph/config:/etc/sourcegraph`, follow these steps:

1. Create files at `$HOME/.sourcegraph/config/ssh/config`, `$HOME/.sourcegraph/config/ssh/known_hosts`, etc., on the host machine as desired to configure SSH.
1. Start (or restart) the container.

To configure the container to use the same SSH as your user account on the host machine, you can also run `cp -R $HOME/.ssh $HOME/.sourcegraph/config/ssh`.

### HTTP(S) authentication via netrc

The easiest way to specify HTTP(S) authentication for repositories is to include the username and password in the clone URL itself, such as `https://user:password@example.com/my/repo`. These credentials won't be displayed to non-admin users.

Otherwise, the container consults the `$HOME/.netrc` files on its own file system for HTTP(S) authentication. The `.netrc` file is a standard way to specify authentication used to connect to external hosts.

To provide HTTP(S) authentication, assuming you're using the default `--volume $HOME/.sourcegraph/config:/etc/sourcegraph`, follow these steps:

1. Create a file at `$HOME/.sourcegraph/config/netrc` on the host machine that contains lines of the form `machine example.com login alice password mypassword` (replacing `example.com`, `alice`, and `mypassword` with the actual values).
1. Start (or restart) the container.

## For Docker Compose deployments

## SSH authentication (config, keys, `known_hosts`)

Provide your `gitserver` instance with your SSH / Git configuration (e.g. `.ssh/id_rsa`, `.ssh/id_rsa.pub`, and `.ssh/known_hosts`--but you can also provide other files like `.netrc`, `.gitconfig`, etc. if needed) by mounting a directory that contains this configuration into the `gitserver` container.

For example, in the `gitserver-0` container configuration in your docker-compose.yaml file, add the second volume listed below, replacing `~/path/on/host/` with the path on the host machine to the `.ssh` directory:

```
gitserver-0:
  container_name: gitserver-0
  ...
  volumes:
    - 'gitserver-0:/data/repos'
    - '~/path/on/host/.ssh:/home/sourcegraph/.ssh'
  ...
```

### HTTP(S) authentication via netrc

The easiest way to specify HTTP(S) authentication for repositories is to include the username and password in the clone URL itself, such as `https://user:password@example.com/my/repo`. These credentials won't be displayed to non-admin users.

Otherwise, follow the steps above for mounting SSH configuration to mount a host directory containing the desired `.netrc` file to `/home/sourcegraph/` in the `gitserver` container.

## For Kubernetes cluster deployments

See "[Configure repository cloning via SSH
](../install/kubernetes/configure.md#configure-repository-cloning-via-ssh)" in the Kubernetes cluster administrator guide.

## For pure-Docker cluster deployments

See "[Configuring SSH cloning](https://github.com/sourcegraph/deploy-sourcegraph-docker/blob/master/README.md#configuring-ssh-cloning)" in the Pure-Docker Sourcegraph cluster deployment reference.
