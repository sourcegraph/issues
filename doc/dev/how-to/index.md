# How to guides

## Local development

- [How to debug live code](debug_live_code.md)
- [Set up local development with Zoekt and Sourcegraph](zoekt_local_dev.md)

## [Troubleshooting](troubleshooting_local_development.md)

- [Problems with node_modules or Javascript packages](troubleshooting_local_development.md#problems-with-nodemodules-or-javascript-packages)
- [dial tcp 127.0.0.1:3090: connect: connection refused](troubleshooting_local_development.md#dial-tcp-1270013090-connect-connection-refused)
- [Database migration failures](troubleshooting_local_development.md#database-migration-failures)
- [Internal Server Error](troubleshooting_local_development.md#internal-server-error)
- [Increase maximum available file descriptors.](troubleshooting_local_development.md#increase-maximum-available-file-descriptors)
- [Caddy 2 certificate problems](troubleshooting_local_development.md#caddy-2-certificate-problems)
- [Running out of disk space](troubleshooting_local_development.md#running-out-of-disk-space)
- [Certificate expiry](troubleshooting_local_development.md#certificate-expiry)
- [CPU/RAM/bandwidth/battery usage](troubleshooting_local_development.md#cpurambandwidthbattery-usage)

## Implementing Sourcegraph

- [Developing the product documentation](documentation_implementation.md)

## Testing Sourcegraph

- [How to run tests](../background-information/testing.md)
- [Configure a test instance of Phabricator and Gitolite](configure_phabricator_gitolite.md)
- [Test a Phabricator and Gitolite instance](test_phabricator.md)

## Windows support

Running Sourcegraph on Windows is not actively tested, but should be possible within the Windows Subsystem for Linux (WSL).
Sourcegraph currently relies on Unix specifics in several places, which makes it currently not possible to run Sourcegraph directly inside Windows without WSL.
We are happy to accept contributions here! :)

## Offline development

Sometimes you will want to develop Sourcegraph but it just so happens you will be on a plane or a
train or perhaps a beach, and you will have no WiFi. And you may raise your fist toward heaven and
say something like, "Why, we can put a man on the moon, so why can't we develop high-quality code
search without an Internet connection?" But lower your hand back to your keyboard and fret no
further, for the year is 2019, and you *can* develop Sourcegraph with no connectivity by setting the
`OFFLINE` environment variable:

```bash
OFFLINE=true dev/start.sh
```
