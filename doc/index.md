# Sourcegraph documentation

[Sourcegraph](https://about.sourcegraph.com) is a web-based, self-hosted code search and navigation tool for developers, used by Uber, Lyft, Yelp, and more.

## Quickstart guide

It takes less than 5 minutes to run and install Sourcegraph using Docker:

<!--
  DO NOT CHANGE THIS TO A CODEBLOCK.
  We want line breaks for readability, but backslashes to escape them do not work cross-platform.
  This uses line breaks that are rendered but not copy-pasted to the clipboard.
-->

<pre class="pre-wrap start-sourcegraph-command"><code>docker run<span class="virtual-br"></span> --publish 7080:7080 --publish 127.0.0.1:3370:3370 --rm<span class="virtual-br"></span> --volume ~/.sourcegraph/config:/etc/sourcegraph<span class="virtual-br"></span> --volume ~/.sourcegraph/data:/var/opt/sourcegraph<span class="virtual-br"></span> sourcegraph/server:3.13.0</code></pre>

Once the server is ready (logo is displayed in the terminal), navigate to the hostname or IP address on port `7080`.  Create the admin account, then you'll be guided through setting up Sourcegraph for code searching and navigation.

<!--
TODO(ryan): Replace with updated screencast
<p class="container">
  <div style="padding:56.25% 0 0 0;position:relative;">
    <iframe src="https://player.vimeo.com/video/314926561?color=0CB6F4&title=0&byline=0&portrait=0" style="position:absolute;top:0;left:0;width:100%;height:100%;" frameborder="0" webkitallowfullscreen mozallowfullscreen allowfullscreen></iframe>
  </div>
</p>
-->

For next steps and further configuration options, visit the [site administration documentation](admin/index.md).

> NOTE: If you get stuck or need help, [file an issue](https://github.com/sourcegraph/sourcegraph/issues/new?&title=Improve+Sourcegraph+quickstart+guide), [tweet (@srcgraph)](https://twitter.com/srcgraph) or [email](mailto:support@sourcegraph.com?subject=Sourcegraph%20quickstart%20guide).

<span class="virtual-br"></span>

> NOTE: If you run Docker on an OS such as RHEL, Fedora, or CentOS with SELinux enabled, sVirt doesn't allow the Docker process
to access `~/.sourcegraph/config` and `~/.sourcegraph/data`. In that case, you will see the following message:

> `Failed to setup nginx:failed to generate nginx configuration to /etc/sourcegraph: open /etc/sourcegraph/nginx.conf: permission denied`.

> To fix this, run:

> `mkdir -p ~/.sourcegraph/config ~/.sourcegraph/data && chcon -R -t svirt_sandbox_file_t ~/.sourcegraph/config ~/.sourcegraph/data`

## Upgrading Sourcegraph

All you need to do to upgrade Sourcegraph is to restart your Docker server with a new image tag.

We actively maintain the two most recent monthly releases of Sourcegraph.

Upgrades should happen across consecutive minor versions of Sourcegraph. For example, if you are
running Sourcegraph 3.1 and want to upgrade to 3.3, you should upgrade to 3.2 and then 3.3.

> The Docker server image tags follow SemVer semantics, so version 3.13.0 can be found at `sourcegraph/server:3.13.0`. You can see the full list of tags on our [Docker Hub page](https://hub.docker.com/r/sourcegraph/server/tags).

## Documentation

Sourcegraph development is open source at [github.com/sourcegraph/sourcegraph](https://github.com/sourcegraph/sourcegraph). Need help? Use the [issue tracker](https://github.com/sourcegraph/sourcegraph/issues).

### Core documentation

- [**User documentation**](user/index.md)
- [**Administrator documentation**](admin/index.md)
- [Install Sourcegraph](admin/install/index.md) or [update Sourcegraph](admin/updates.md)
- [Sourcegraph extensions](extensions/index.md)
- [Product direction (roadmap)](https://about.sourcegraph.com/direction)

### Features and tutorials

- [Overview](user/index.md): What is Sourcegraph?
- [Tour](user/tour.md): A walkthrough of Sourcegraph's features, with real-world example use cases.
- [How to run a Sourcegraph trial](adopt/trial/index.md) at your company
- [Integrations](integration/index.md) with GitHub, GitLab, Bitbucket, etc.
- [Chrome and Firefox browser extensions](integration/browser_extension.md)
- [Query syntax reference](user/search/queries.md)
- [GraphQL API](api/graphql/index.md)
- [Sourcegraph Enterprise](admin/subscriptions/index.md)

## Sourcegraph subscriptions

You can use Sourcegraph in 2 ways:

- [Self-hosted Sourcegraph](admin/install/index.md): Deploy and manage your own Sourcegraph instance.
- [Sourcegraph.com](https://sourcegraph.com): For public code only. No signup or installation required.

For self-hosted Sourcegraph instances, you run a Docker image or Kubernetes cluster on-premises or on your preferred cloud provider. There are [2 tiers](https://about.sourcegraph.com/pricing): Core (free) and Enterprise. Enterprise features require a [Sourcegraph subscription](https://about.sourcegraph.com/contact/sales).

## Other links

- [Sourcegraph open-source repository](https://github.com/sourcegraph/sourcegraph)
- [Contributing to Sourcegraph](dev/index.md)
- [Sourcegraph handbook](https://about.sourcegraph.com/handbook)
- [Sourcegraph blog](https://about.sourcegraph.com/blog/)
- [Issue tracker](https://github.com/sourcegraph/sourcegraph/issues)
- [about.sourcegraph.com](https://about.sourcegraph.com) (general information about Sourcegraph)
- [@srcgraph on Twitter](https://twitter.com/srcgraph)
