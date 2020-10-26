# Custom git or ssh config

Sourcegraph supports customising [git-config](https://git-scm.com/docs/git-config) and [ssh_config](https://linux.die.net/man/5/ssh_config) for adjusting the behaviour of git. Sourcegraph will read these from the standard locations. This guide documents how to do it for each target environment and common use cases.

## Setting configuration

For cluster environments, we have guides for configuring SSH cloning. These can be adapted to additionally set `/etc/gitconfig`:
- Kubernetes guide to [configure repository cloning via SSH](../install/kubernetes/configure.md#configure-repository-cloning-via-ssh).
- Docker guide to [configure SSH cloning](https://github.com/sourcegraph/deploy-sourcegraph-docker/blob/master/README.md#configuring-ssh-cloning)

Upon the Sourcegraph Docker image container start, it copies all files from `/etc/sourcegraph/{ssh,gitconfig,netrc}` into its own `$HOME` directory. Alternatively you can create a new Docker image which inherits from Sourcegraph and then mutates the environment:

``` dockerfile
FROM sourcegraph/server:3.21.2

COPY gitconfig /etc/gitconfig
COPY ssh /root/.ssh
RUN	find /root/.ssh -type f -exec chmod 600 '{}' ';'
RUN	find /root/.ssh -type d -exec chmod 700 '{}' ';'
```

This approach can also be used for `sourcegraph/gitserver` images in cluster environments.

## Example: alternate clone URL for repos

Some sites put an HTTPS or SSH proxy in front of their code host to reduce load. Some sites also use a service like AWS CodeCommit to do the same. In these cases, the repos still should be treated as being repos on the original code host, not the proxy site.

For example, I have a GitHub repo `github.com/foo/bar`. I want Sourcegraph to clone it using the URL https://cloneproxy.example.com/foo/bar.git. But I still want the "Go to GitHub repository" button, etc., to take me to https://github.com/foo/bar. You can use the git configuration [`insteadOf`](https://git-scm.com/docs/git-config#Documentation/git-config.txt-urlltbasegtinsteadOf):

``` ini
# ~/.gitconfig or /etc/gitconfig
[url "https://cloneproxy.example.com"]
  insteadOf = https://github.com
```

If you are cloning via SSH, you can also achieve this with an SSH configuration:

```
# ~/.ssh/config
Host github.com
  Hostname cloneproxy.example.com
```
