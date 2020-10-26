# Troubleshooting

## Problems with node_modules or Javascript packages

Noticing problems with <code>node_modules/</code> or package versions? Try
running this command to clear the local package cache.

```bash
yarn cache clean
rm -rf node_modules
yarn
```

## Node version out of date

```bash
Validating package.json...
error @: The engine "node" is incompatible with this module. Expected version "^v14.7.0". Got "14.5.0"
```
If you see an error like this you need to upgrade the version of node installed. You can do this with `nvm use` and then following the prompts to install or update to the correct Node.js version.

### dial tcp 127.0.0.1:3090: connect: connection refused

This means the `frontend` server failed to start, for some reason. Look through
the previous logs for possible explanations, such as failure to contact the
`redis` server, or database migrations failing.

### Database migration failures

While developing Sourcegraph, you may run into:

`frontend | failed to migrate the DB. Please contact hi@sourcegraph.com for further assistance:Dirty database version 1514702776. Fix and force version.`

You may have to run migrations manually. First, install the Go [migrate](https://github.com/golang-migrate/migrate/tree/master/cli#installation) CLI, then run `dev/db/migrate.sh <db_name> up` where the database name is either `frontend` or `codeintel`.

If you get something like `error: Dirty database version 1514702776. Fix and force version.`, you need to roll things back and start from scratch.

```bash
dev/db/migrate.sh <db_name> drop
dev/db/migrate.sh <db_name> up
```

If you receive errors while migrating, try dropping the database

```bash
dev/db/drop-entire-local-database-and-redis.sh
dev/db/migrate.sh <db_name> up
```

### Internal Server Error

If you see this error when opening the app:

`500 Internal Server Error template: app.html:21:70: executing "app.html" at <version "styles/styl...>: error calling version: open ui/assets/styles/app.bundle.css: no such file or directory`

that means Webpack hasn't finished compiling the styles yet (it takes about 3 minutes).
Simply wait a little while for a message from webpack like `web | Time: 180000ms` to appear
in the terminal.

### Increase maximum available file descriptors.

`./dev/start.sh` may ask you to run ulimit to increase the maximum number
of available file descriptors for a process. You can make this setting
permanent for every shell session by adding the following line to your
`.*rc` file (usually `.bashrc` or `.zshrc`):

```bash
# increase max number of file descriptors for running a sourcegraph instance.
ulimit -n 10000
```

On Linux, it may also be necessary to increase `sysctl -n fs.inotify.max_user_watches`, which can be
done by running one of the following:

```bash
echo 524288 | sudo tee -a /proc/sys/fs/inotify/max_user_watches

# If the above doesn't work, you can also try this:
sudo sysctl fs.inotify.max_user_watches=524288
```

If you ever need to wipe your local database and Redis, run the following command.

```bash
./dev/drop-entire-local-database-and-redis.sh
```

### Caddy 2 certificate problems

We use Caddy 2 to setup HTTPS for local development. It creates self-signed certificates and uses that to serve the local Sourcegraph instance. If your browser complains about the certificate, check the following:

1. The first time that Caddy 2 reverse-proxies your Sourcegraph instance, it needs to add its certificate authority to your local certificate store. This may require elevated permissions on your machine. If you haven't done so already, try running `caddy reverse-proxy --to localhost:3080` and enter your password if prompted. You may also need to run that command as the `root` user.

1. If you have completed the previous step and your browser still complains about the certificate, try restarting your browser or your local machine.

#### Adding Caddy certificates to Windows

When running Caddy on WSL, you need to manually add the Caddy root certificate to the Windows certificate store using [certutil.exe](https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/certutil).

```bash
# Run inside WSL
certutil.exe -addstore -user Root "$(find /usr/local/share/ca-certificates/ -name '*Caddy*')"
```

This command will add the certificate to the `Trusted Root Certification Authorities` for your Windows user.

### Running out of disk space

If you see errors similar to this:

```
gitserver | ERROR cleanup: error freeing up space, error: only freed 1124101958 bytes, wanted to free 29905298227
```

You are probably low on disk space. By default it tries to cleanup when there is less than 10% of available disk space.
You can override that by setting this env variable:

```bash
# means 5%. You may want to put that into .bashrc for convinience
SRC_REPOS_DESIRED_PERCENT_FREE=5
```

### Certificate expiry

If you see a certificate expiry warning you may need to delete your certificate and restart your server.

On MaCOS, the certificate can be removed from here: `~/Library/Application\ Support/Caddy/certificates/local/sourcegraph.test`

## CPU/RAM/bandwidth/battery usage

On first install, the program will use quite a bit of bandwidth to concurrently
download all of the Go and Node packages. After packages have been installed,
the Javascript assets will be compiled into a single Javascript file, which
can take up to 5 minutes, and can be heavy on the CPU at times.

After the initial install/compile is complete, the Docker for Mac binary uses
about 1.5GB of RAM. The numerous different Go binaries don't use that much RAM
or CPU each, about 5MB of RAM each.

If you notice heavy battery and CPU usage running `gulp --color watch`, please first [double check that Spotlight is not indexing your Sourcegraph repository](https://www.macobserver.com/tips/how-to/stop-spotlight-indexing/), as this can lead to additional, unnecessary, poll events.

If you're running macOS 10.15.x (Catalina) reinstalling the Xcode Command Line Tools may be necessary as follows:

1. Uninstall the Command Line Tools with `rm -rf /Library/Developer/CommandLineTools`
2. Reinstall it with `xcode-select --install`
3. Go to `sourcegraph/sourcegraph`’s root directory and run `rm -rf node_modules`
3. Re-run the launch script (`./dev/start.sh`)
