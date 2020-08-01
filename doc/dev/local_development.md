<!-- omit in toc -->
# Getting started with developing Sourcegraph

Have a look around, our code is on [GitHub](https://sourcegraph.com/github.com/sourcegraph/sourcegraph).

<!-- omit in toc -->
## Outline

- [Environment](#environment)
- [Step 1: Install dependencies](#step-1-install-dependencies)
  - [macOS](#macos)
  - [Ubuntu](#ubuntu)
  - [(optional) asdf](#optional-asdf)
- [Step 2: Initialize your database](#step-2-initialize-your-database)
  - [More info](#more-info)
- [Step 3: (macOS) Start Docker](#step-3-macos-start-docker)
- [Step 4: Get the code](#step-4-get-the-code)
- [Step 5: Configure HTTPS reverse proxy](#step-5-configure-https-reverse-proxy)
  - [Prerequisites](#prerequisites)
    - [Add `sourcegraph.test` to `/etc/hosts`](#add-sourcegraphtest-to-etchosts)
    - [Initialize Caddy 2](#initialize-caddy-2)
- [Step 6: Start the server](#step-6-start-the-server)
- [Troubleshooting](#troubleshooting)
    - [Problems with node_modules or Javascript packages](#problems-with-nodemodules-or-javascript-packages)
    - [dial tcp 127.0.0.1:3090: connect: connection refused](#dial-tcp-1270013090-connect-connection-refused)
    - [Database migration failures](#database-migration-failures)
    - [Internal Server Error](#internal-server-error)
    - [Increase maximum available file descriptors.](#increase-maximum-available-file-descriptors)
    - [Caddy 2 certificate problems](#caddy-2-certificate-problems)
    - [Running out of disk space](#running-out-of-disk-space)
    - [Certificate expiry](#certificate-expiry)
- [How to Run Tests](#how-to-run-tests)
- [CPU/RAM/bandwidth/battery usage](#cpurambandwidthbattery-usage)
- [How to debug live code](#how-to-debug-live-code)
  - [Debug TypeScript code](#debug-typescript-code)
  - [Debug Go code](#debug-go-code)
- [Go dependency management](#go-dependency-management)
- [Codegen](#codegen)
- [Windows support](#windows-support)
- [Other nice things](#other-nice-things)
  - [Offline development](#offline-development)

## Environment

Sourcegraph server is a collection of smaller binaries. The development server, [dev/start.sh](https://github.com/sourcegraph/sourcegraph/blob/master/dev/start.sh), initializes the environment and starts a process manager that runs all of the binaries. See the [Architecture doc](architecture/index.md) for a full description of what each of these services does. The sections below describe the dependencies you need to run `dev/start.sh`.

<!-- omit in toc -->
### For Sourcegraph employees

You'll need to clone [`sourcegraph/dev-private`](https://github.com/sourcegraph/dev-private) (which has convenient preconfigured settings and external services on an enterprise account) alongside the `sourcegraph/sourcegraph` repository, for example:

```
/dir
 |-- dev-private
 +-- sourcegraph
```

After the initial setup you can run `enterprise/dev/start.sh` instead of `dev/start.sh`.

## Step 1: Install dependencies


> NOTE: Please see install instructions for [macOS](#macos) and [Ubuntu](#ubuntu) in succeeding sections.

Sourcegraph has the following dependencies:
- [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git) (v2.18 or higher)
- [Go](https://golang.org/doc/install) (v1.14 or higher)
- [Node JS](https://nodejs.org/en/download/) (see current recommended version in [.nvmrc](https://github.com/sourcegraph/sourcegraph/blob/master/.nvmrc))
- [make](https://www.gnu.org/software/make/)
- [Docker](https://docs.docker.com/engine/installation/) (v18 or higher)
  - For macOS we recommend using Docker for Mac instead of `docker-machine`
- [PostgreSQL](https://wiki.postgresql.org/wiki/Detailed_installation_guides) (v11 or higher)
- [Redis](http://redis.io/) (v5.0.7 or higher)
- [Yarn](https://yarnpkg.com) (v1.10.1 or higher)
- [NGINX](https://docs.nginx.com/nginx/admin-guide/installing-nginx/installing-nginx-open-source/) (v1.14 or higher)
- [SQLite](https://www.sqlite.org/index.html) tools
- [Golang Migrate](https://github.com/golang-migrate/migrate/) (v4.7.0 or higher)
- [Comby](https://github.com/comby-tools/comby/) (v0.11.3 or higher)
- [Watchman](https://facebook.github.io/watchman/)

The following are two recommendations for installing these dependencies:

### macOS

1.  Install [Homebrew](https://brew.sh).
2.  Install [Docker for Mac](https://docs.docker.com/docker-for-mac/).

    optionally via `brew`

    ```
    brew cask install docker
    ```

3.  Install Go, Node Version Manager, PostgreSQL, Redis, Git, NGINX, golang-migrate, Comby, SQLite tools, and jq with the following command:

    ```
    brew install go yarn redis postgresql git gnu-sed nginx golang-migrate comby sqlite pcre FiloSottile/musl-cross/musl-cross jq watchman
    ```

4.  Install the Node Version Manager (`nvm`) using:

    ```
    NVM_VERSION="$(curl https://api.github.com/repos/nvm-sh/nvm/releases/latest | jq -r .name)"
    curl -L https://raw.githubusercontent.com/nvm-sh/nvm/"$NVM_VERSION"/install.sh -o install-nvm.sh
    sh install-nvm.sh
    ```

    After the install script is finished, re-source your shell profile (e.g.,
    `source ~/.zshrc`) or restart your terminal session to pick up the `nvm`
    definitions. Re-running the install script will update the installation.

    Note: `nvm` is implemented as a shell function, so it may not show up in
    the output of `which nvm`. Use `type nvm` to verify whether it is set up.
    There is also a Homebrew package for `nvm`, but it is unsupported by the
    `nvm` maintainers.

5.  Install the current recommended version of Node JS by running the following
    from the working directory of a sourcegraph repository clone:

    ```
    nvm install
    nvm use --delete-prefix
    ```

    After doing this, `node -v` should show the same version mentioned in
    `.nvmrc` at the root of the sourcegraph repository.

    Note: Although there is a Homebrew package for Node, we advise using `nvm`
    instead, to ensure you get a Node version compatible with the current state
    of the sourcegraph repository.

6.  Configure PostgreSQL and Redis to start automatically

    ```
    brew services start postgresql
    brew services start redis
    ```

    (You can stop them later by calling `stop` instead of `start` above.)

7.  Ensure `psql`, the PostgreSQL command line client, is on your `$PATH`.
    Homebrew does not put it there by default. Homebrew gives you the command to run to insert `psql` in your path in the "Caveats" section of `brew info postgresql`. Alternatively, you can use the command below. It might need to be adjusted depending on your Homebrew prefix (`/usr/local` below) and shell (bash below).

    ```
    hash psql || { echo 'export PATH="/usr/local/opt/postgresql/bin:$PATH"' >> ~/.bash_profile }
    source ~/.bash_profile
    ```

8.  Open a new Terminal window to ensure `psql` is now on your `$PATH`.

### Ubuntu


1. Add package repositories:

    ```
    # Go
    sudo add-apt-repository ppa:longsleep/golang-backports

    # Docker
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
    sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"

    # Yarn
    curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | sudo apt-key add -
    echo "deb https://dl.yarnpkg.com/debian/ stable main" | sudo tee /etc/apt/sources.list.d/yarn.list
    ```

2. Update repositories:

    ```
    sudo apt-get update
    ```

3. Install dependencies:

    ```
    sudo apt install -y make git-all postgresql postgresql-contrib redis-server nginx libpcre3-dev libsqlite3-dev pkg-config golang-go musl-tools docker-ce docker-ce-cli containerd.io yarn jq

    # install golang-migrate (you must rename the extracted binary to `golang-migrate` and move the binary into your $PATH)
    curl -L https://github.com/golang-migrate/migrate/releases/download/v4.7.0/migrate.linux-amd64.tar.gz | tar xvz

    # install comby (you must rename the extracted binary to `comby` and move the binary into your $PATH)
    curl -L https://github.com/comby-tools/comby/releases/download/0.11.3/comby-0.11.3-x86_64-linux.tar.gz | tar xvz

    # install watchman (you must put the binary and shared libraries on your $PATH and $LD_LIBRARY_PATH)
    curl -L https://github.com/facebook/watchman/releases/download/v2020.07.13.00/watchman-v2020.07.13.00-linux.zip

    # nvm (to manage Node.js)
    NVM_VERSION="$(curl https://api.github.com/repos/nvm-sh/nvm/releases/latest | jq -r .name)"
    curl -L https://raw.githubusercontent.com/nvm-sh/nvm/"$NVM_VERSION"/install.sh -o install-nvm.sh
    sh install-nvm.sh

    # in repo dir: install current recommendend version of Node JS
    nvm install
    ```

4. Configure startup services

    ```
    sudo systemctl enable postgresql
    sudo systemctl enable redis-server.service
    ```

5. (optional) You can also run Redis using Docker

    In this case you should not enable the `redis-server.service` from the previous step.

    ```
    dockerd # if docker isn't already running
    docker run -p 6379:6379 -v $REDIS_DATA_DIR redis
    # $REDIS_DATA_DIR should be an absolute path to a folder where you intend to store Redis data
    ```

    You need to have Redis running when you start the dev server later on. If you have issues running Docker, try [adding your user to the docker group][dockerGroup], and/or [updating the socket file persimissions][socketPermissions], or try running these commands under `sudo`.

    [dockerGroup]: https://stackoverflow.com/a/48957722
    [socketPermissions]: https://stackoverflow.com/a/51362528

### (optional) asdf

[asdf](https://github.com/asdf-vm/asdf) is a CLI tool that manages runtime versions for a number of different languages and tools. It can be likened to a language-agnostic version of [nvm](https://github.com/nvm-sh/nvm) or [pyenv](https://github.com/pyenv/pyenv).

We use asdf in buildkite to lock the versions of the tools that we use on a per-commit basis.

<!-- omit in toc -->
#### Install

<!-- omit in toc -->
##### asdf binary

See the [installation instructions on the official asdf documentation](https://asdf-vm.com/#/core-manage-asdf-vm?id=install-asdf-vm).

<!-- omit in toc -->
##### Plugins

sourcegraph/sourcegraph uses the following plugins:

- [Go](https://github.com/kennyp/asdf-golang)

```bash
asdf plugin add golang
```

- [NodeJS](https://github.com/asdf-vm/asdf-nodejs)

```bash
asdf plugin add nodejs

# Import the Node.js release team's OpenPGP keys to main keyring
bash ~/.asdf/plugins/nodejs/bin/import-release-team-keyring

# Have asdf read .nvmrc for auto-switching between node version
## Add the following to $HOME/.asdfrc:
legacy_version_file = yes
```

- [Yarn](https://github.com/twuni/asdf-yarn)

```bash
asdf plugin add yarn
```

<!-- omit in toc -->
#### Usage instructions

[asdf](https://github.com/asdf-vm/asdf) uses versions specified in [.tool-versions](https://github.com/sourcegraph/sourcegraph/blob/master/.tool-versions) whenever a command is run from one of `sourcegraph/sourcegraph`'s subdirectories.

You can install the all the versions specified in [.tool-versions](https://github.com/sourcegraph/sourcegraph/blob/master/.tool-versions) by running `asdf install`.


## Step 2: Initialize your database

### Without Docker

You need a fresh Postgres database and a database user that has full ownership of that database.

1. Create a database for the current Unix user

    ```
    # For Linux users, first access the postgres user shell
    sudo su - postgres
    ```

    ```
    createdb
    ```

2. Create the Sourcegraph user and password

    ```
    createuser --superuser sourcegraph
    psql -c "ALTER USER sourcegraph WITH PASSWORD 'sourcegraph';"
    ```

3. Create the Sourcegraph database

    ```
    createdb --owner=sourcegraph --encoding=UTF8 --template=template0 sourcegraph
    ```

4. Configure database settings in your environment

    The Sourcegraph server reads PostgreSQL connection configuration from the [`PG*` environment variables](http://www.postgresql.org/docs/current/static/libpq-envars.html).

    Add these, for example, in your `~/.bashrc`:

    ```
    export PGPORT=5432
    export PGHOST=localhost
    export PGUSER=sourcegraph
    export PGPASSWORD=sourcegraph
    export PGDATABASE=sourcegraph
    export PGSSLMODE=disable
    ```

    You can also use a tool like [`envdir`][envdir] or [a `.dotenv` file][dotenv] to
    source these env vars on demand when you start the server.

    [envdir]: https://cr.yp.to/daemontools/envdir.html
    [dotenv]: https://github.com/joho/godotenv

### With Docker

You may also want to run Postgres within a docker container instead of as a system service. Running within a container provides some advantages such as storing the data seperately from the container, you do not need to run it as a system service and its easy to use different database versions or multiple databases.

1. Create a directory to store and mount the database from for persistence:

    ```
    # Create a seperate dir to store the database
    mkdir PGDATA_DIR

   # Also add this to your '~/.bashrc'
    export PGDATA_DIR=/path/to/PGDATA_DIR/
    ```

2. Run the container:

  ```
   docker run -d  -p 5432:5432 -e POSTGRES_PASSWORD=sourcegraph \
   -e POSTGRES_USER=sourcegraph -e POSTGRES_INITDB_ARGS=" --encoding=UTF8 " \
   -v $PGDATA_DIR:/var/lib/postgresql/data postgres
   ```

3. Ensure you can connect to the database using `psql -U sourcegraph` and enter password `sourcegraph`.

4. Configure database settings in your environment:

    The Sourcegraph server reads PostgreSQL connection configuration from the [`PG*` environment variables](http://www.postgresql.org/docs/current/static/libpq-envars.html).

    Add these, for example, in your `~/.bashrc`:

    ```
    export PGPORT=5432
    export PGHOST=localhost
    export PGUSER=sourcegraph
    export PGPASSWORD=sourcegraph
    export PGDATABASE=sourcegraph
    export PGSSLMODE=disable
    ```

    You can also use a tool like [`envdir`][envdir] or [a `.dotenv` file][dotenv] to source these env vars on demand when you start the server.

    [envdir]: https://cr.yp.to/daemontools/envdir.html
    [dotenv]: https://github.com/joho/godotenv

5. On restarting docker, you may need to start the container again. Find the image with `docker ps --all` and then `docker run <$containerID>` to start again.

### More info

For more information about data storage, [read our full PostgreSQL Guide
page](postgresql.md).

Migrations are applied automatically.

## Step 3: (macOS) Start Docker

<!-- omit in toc -->
#### Option A: Docker for Mac

This is the easy way - just launch Docker.app and wait for it to finish loading.

<!-- omit in toc -->
#### Option B: docker-machine

The Docker daemon should be running in the background, which you can test by
running `docker ps`. If you're on OS X and using `docker-machine` instead of
Docker for Mac, you may have to run:

```bash
docker-machine start default
eval $(docker-machine env)
```

## Step 4: Get the code

```bash
git clone https://github.com/sourcegraph/sourcegraph.git
```

## Step 5: Configure HTTPS reverse proxy

Sourcegraph's development environment ships with a [Caddy 2](https://caddyserver.com/) HTTPS reverse proxy that allows you to access your local sourcegraph instance via `https://sourcegraph.test:3443` (a fake domain with a self-signed certificate that's added to `/etc/hosts`).

If you'd like Sourcegraph to be accessible under `https://sourcegraph.test` (port 443) instead, you can [set up authbind](https://medium.com/@steve.mu.dev/setup-authbind-on-mac-os-6aee72cb828) and set the environment variable `SOURCEGRAPH_HTTPS_PORT=443`.

### Prerequisites

In order to configure the HTTPS reverse-proxy, you'll need to edit `/etc/hosts` and initialize Caddy 2.

#### Add `sourcegraph.test` to `/etc/hosts`

`sourcegraph.test` needs to be added to `/etc/hosts` as an alias to `127.0.0.1`. There are two main ways of accomplishing this:

1. Manually append `127.0.0.1 sourcegraph.test` to `/etc/hosts`
1. Use the provided `./dev/add_https_domain_to_hosts.sh` convenience script (sudo may be required).

```bash
> ./dev/add_https_domain_to_hosts.sh

--- adding sourcegraph.test to '/etc/hosts' (you may need to enter your password)
Password:
Adding host(s) "sourcegraph.test" to IP address 127.0.0.1
--- printing '/etc/hosts'
...
127.0.0.1        localhost sourcegraph.test
...
```

#### Initialize Caddy 2

[Caddy 2](https://caddyserver.com/) automatically manages self-signed certificates and configures your system so that your web browser can properly recognize them. The first time that Caddy runs, it needs `root/sudo` permissions to add
its keys to your system's certificate store. You can get this out the way after installing Caddy 2 by running the following command and entering your password if prompted:

```bash
./dev/caddy.sh trust
```

You might need to restart your web browsers in order for them to recognize the certificates.

## Step 6: Start the server

```bash
cd sourcegraph
./dev/start.sh
```

This will continuously compile your code and live reload your locally running instance of Sourcegraph.

Navigate your browser to https://sourcegraph.test:3443 to see if everything worked.

## Troubleshooting

### Problems with node_modules or Javascript packages

Noticing problems with <code>node_modules/</code> or package versions? Try
running this command to clear the local package cache.

```bash
yarn cache clean
rm -rf node_modules web/node_modules
yarn
cd web
yarn
```

### dial tcp 127.0.0.1:3090: connect: connection refused

This means the `frontend` server failed to start, for some reason. Look through
the previous logs for possible explanations, such as failure to contact the
`redis` server, or database migrations failing.

### Database migration failures

While developing Sourcegraph, you may run into:

`frontend | failed to migrate the DB. Please contact hi@sourcegraph.com for further assistance:Dirty database version 1514702776. Fix and force version.`

You may have to run migrations manually. First, install the Go [migrate](https://github.com/golang-migrate/migrate/tree/master/cli#installation) CLI, then run `dev/migrate.sh up`

If you get something like `error: Dirty database version 1514702776. Fix and force version.`, you need to roll things back and start from scratch.

```bash
dev/migrate.sh drop
dev/migrate.sh up
```

If you receive errors while migrating, try dropping the database

```bash
dev/drop-entire-local-database.sh
dev/migrate.sh up
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

#### Caddy 2 certificate problems

We use Caddy 2 to setup HTTPS for local development. It creates self-signed certificates and uses that to serve the local Sourcegraph instance. If your browser complains about the certificate, check the following:

1. The first time that Caddy 2 reverse-proxies your Sourcegraph instance, it needs to add its certificate authority to your local certificate store. This may require elevated permissions on your machine. If you haven't done so already, try running `caddy reverse-proxy --to localhost:3080` and enter your password if prompted. You may also need to run that command as the `root` user.

1. If you have completed the previous step and your browser still complains about the certificate, try restarting your browser or your local machine.

##### Adding Caddy certificates to Windows

When running Caddy on WSL, you need to manually add the Caddy root certificate to the Windows certificate store using [certutil.exe](https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/certutil).

```bash
# Run inside WSL
certutil.exe -addstore -user Root "$(find /usr/local/share/ca-certificates/ -name '*Caddy*')"
```

This command will add the certificate to the `Trusted Root Certification Authorities` for your Windows user.

#### Running out of disk space

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

#### Certificate expiry

If you see a certificate expiry warning you may need to delete your certificate and restart your server.

On MaCOS, the certificate can be removed from here: `~/Library/Application\ Support/Caddy/certificates/local/sourcegraph.test`

## How to Run Tests

See [testing.md](testing.md) for details.

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

## How to debug live code

How to debug a program with Visual Studio Code:

### Debug TypeScript code

Requires "Debugger for Chrome" extension.

- Quit Chrome
- Launch Chrome (Canary) from the command line with a remote debugging port:
  - Mac OS: `/Applications/Google\ Chrome\ Canary.app/Contents/MacOS/Google\ Chrome\ Canary --remote-debugging-port=9222`
  - Windows: `start chrome.exe –remote-debugging-port=9222`
  - Linux: `chromium-browser --remote-debugging-port=9222`
- Go to http://localhost:3080
- Open the Debugger in VSCode: "View" > "Debug"
- Launch the `(ui) http://localhost:3080/*` debug configuration
- Set breakpoints, enjoy

### Debug Go code

Install [Delve](https://github.com/derekparker/delve):

```bash
xcode-select --install
pushd /tmp
go get github.com/go-delve/delve/cmd/dlv
popd /tmp
```

Then install `pgrep`:

```bash
brew install proctools
```

Make sure to run `env DELVE=true dev/start.sh` to disable optimizations during compilation, otherwise Delve will have difficulty stepping through optimized functions (line numbers will be off, you won't be able to print local variables, etc.).

Now you can attach a debugger to any Go process (e.g. frontend, searcher, go-langserver) in 1 command:

```bash
dlv attach $(pgrep frontend)
```

Delve will pause the process once it attaches the debugger. Most used [commands](https://github.com/go-delve/delve/tree/master/Documentation/cli):

- `b internal/db/access_tokens.go:52` to set a breakpoint on a line (`bp` lists all, `clearall` deletes all)
- `c` to continue execution of the program
- `Ctrl-C` pause the program to bring back the command prompt
- `n` to step over the next statement
- `s` to step into the next function call
- `stepout` to step out of the current function call
- `Ctrl-D` to exit

## Go dependency management

We use Go modules to manage Go dependencies in this repository.

## Codegen

The Sourcegraph repository relies on code generation triggered by `go generate`. Code generation is used for a variety of tasks:

- generating code for mocking interfaces
- generate wrappers for interfaces (e.g., `./server/internal/middleware/*` packages)
- pack app templates and assets into binaries

To generate everything, just run:

```bash
./dev/generate.sh
```

Note: Sometimes, there are erroneous diffs. This occurs for a few
reasons, none of which are legitimate (i.e., they are tech debt items
we need to address):

- The codegen tools might emit code that depends on system configuration,
  such as the system timezone or packages you have in your GOPATH. We
  need to submit PRs to the tools to eliminate these issues.
- You might have existing but gitignored files that the codegen tools
  read on your disk that other developers don't have. (This occurs for
  app assets especially.)

If you think a diff is erroneous, don't commit it. Add a tech debt
item to the issue tracker and assign the person who you think is
responsible (or ask).

## Windows support

Running Sourcegraph on Windows is not actively tested, but should be possible within the Windows Subsystem for Linux (WSL).
Sourcegraph currently relies on Unix specifics in several places, which makes it currently not possible to run Sourcegraph directly inside Windows without WSL.
We are happy to accept contributions here! :)

## Other nice things

### Offline development

Sometimes you will want to develop Sourcegraph but it just so happens you will be on a plane or a
train or perhaps a beach, and you will have no WiFi. And you may raise your fist toward heaven and
say something like, "Why, we can put a man on the moon, so why can't we develop high-quality code
search without an Internet connection?" But lower your hand back to your keyboard and fret no
further, for the year is 2019, and you *can* develop Sourcegraph with no connectivity by setting the
`OFFLINE` environment variable:

```bash
OFFLINE=true dev/start.sh
```
