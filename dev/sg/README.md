# sg - the Sourcegraph developer tool

```
          _____                    _____
         /\    \                  /\    \
        /::\    \                /::\    \
       /::::\    \              /::::\    \
      /::::::\    \            /::::::\    \
     /:::/\:::\    \          /:::/\:::\    \
    /:::/__\:::\    \        /:::/  \:::\    \
    \:::\   \:::\    \      /:::/    \:::\    \
  ___\:::\   \:::\    \    /:::/    / \:::\    \
 /\   \:::\   \:::\    \  /:::/    /   \:::\ ___\
/::\   \:::\   \:::\____\/:::/____/  ___\:::|    |
\:::\   \:::\   \::/    /\:::\    \ /\  /:::|____|
 \:::\   \:::\   \/____/  \:::\    /::\ \::/    /
  \:::\   \:::\    \       \:::\   \:::\ \/____/
   \:::\   \:::\____\       \:::\   \:::\____\
    \:::\  /:::/    /        \:::\  /:::/    /
     \:::\/:::/    /          \:::\/:::/    /
      \::::::/    /            \::::::/    /
       \::::/    /              \::::/    /
        \::/    /                \::/____/
         \/____/

```

`sg` is the CLI tool that Sourcegraph developers can use to develop Sourcegraph.

- [Quickstart](#quickstart)
- [Usage](#usage)
  - [`sg [start,run,run-set]` - Start dev environment](#sg-startrunrun-set---start-dev-environment)
  - [`sg test` - Running test suites](#sg-test---running-test-suites)
  - [`sg doctor` - Check health of dev environment](#sg-doctor---check-health-of-dev-environment)
  - [`sg live` - See currently deployed version](#sg-live---see-currently-deployed-version)
  - [`sg migration` - Run or manipulate database migrations](#sg-migration---run-or-manipulate-database-migrations)
- [Configuration](#configuration)
- [TODOs](#todos)
- [Hacking](#hacking)
- [Principles](#principles)
- [Inspiration](#inspiration)
- [Ideas](#ideas)
  - [Generators](#generators)
  - [Edit configuration files](#edit-configuration-files)
  - [Tail logs](#tail-logs)

## Quickstart

Run the following to install `sg` from inside `sourcegraph/sourcegraph`:

```
./dev/sg/install.sh
```

Make sure that `$HOME/go/bin` is in your `$PATH`. (If you use `$GOPATH` then `$GOPATH/bin` needs to be in the `$PATH`)

**Note for Linux users:** A command called [sg](https://www.man7.org/linux/man-pages/man1/sg.1.html) is already available at `/usr/bin/sg`. To use the Sourcegraph `sg` CLI, you need to make sure that its location comes first in `PATH`. For example, by prepending `$GOPATH/bin`:

```
export PATH=$GOPATH/bin:$PATH
```

Instead of the more conventional:

```
export PATH=$PATH:$GOPATH/bin
```

Or you may add an alias to your `.bashrc`:

```
alias sg=$HOME/go/bin/sg
```

Then, in the root of `sourcegraph/sourcegraph`, run:

```
sg start
```

This will boot the `default` commands in `sg.config.yaml` in the root of the repository.

**Alternative install method** (if you want to move the binary to a custom location):

In the root of `sourcegraph/sourcegraph`, run the following:

```
go build -o ~/my/path/sg ./dev/sg
```

Make sure that `~/my/path` is in your `$PATH` then.

## Usage

### `sg [start,run,run-set]` - Start dev environment

```bash
# Run default environment (this starts the 'default' command set defined in `sg.config.yaml`):
sg start

# Run the enterprise environment:
sg run-set enterprise

# Run specific commands:
sg run gitserver
sg run frontend

# List available commands:
sg run -help

# List available command sets:
sg run-set -help
```

### `sg test` - Running test suites

```bash
# Run different test suites:
sg test backend
sg test backend-integration
sg test frontend
sg test frontend-e2e

# List available test suites:
sg test -help

# Arguments are passed along to the command
sg test backend-integration -run TestSearch
```

### `sg doctor` - Check health of dev environment

```bash
# Run the checks defined in sg.config.yaml
sg doctor
```

### `sg live` - See currently deployed version

```bash
# See which version is deployed on a preset environment
sg live dot-com
sg live k8s

# See which version is deployed on a custom environment
sg live https://demo.sourcegraph.com

# List environments:
sg live -help
```

### `sg migration` - Run or manipulate database migrations

```bash
# Migrate local default database up
sg migration up

# Migrate specific database down one migration
sg migration down --db codeintel -n 1

# Add new migration for specific database
sg migration add --db codeintel 'add missing index'

# Squash migrations for default database
sg migration squash

# Fixup your migrations comapred to main for databases
sg migration fixup

# To see what operations `sg migration fixup` will run, you can check with
sg migration fixup -run=false

# Or to run for only one database, you can use the -db flag, as in other operations.
```

## Configuration

`sg` is configured through the `sg.yaml` file in the root of the `sourcegraph/sourcegraph` repository:

```yaml
commands:
  gitserver:
    cmd: .bin/gitserver
    install: go install github.com/sourcegraph/sourcegraph/cmd/gitserver -o .bin/gitserver
    env:
      HOSTNAME: $SRC_GIT_SERVER_1

  enterprise-gitserver:
    cmd: .bin/gitserver
    install: go install github.com/sourcegraph/sourcegraph/enterprise/cmd/gitserver -o .bin/gitserver
    env:
      HOSTNAME: $SRC_GIT_SERVER_1

  query-runner:
    cmd: .bin/query-runner
    install: go install github.com/sourcegraph/sourcegraph/cmd/query-runner -o .bin/gitserver

  searcher:
    cmd: .bin/searcher
    install: go install github.com/sourcegraph/sourcegraph/cmd/searcher -o .bin/gitserver

  caddy:
    installDoc.darwin: 'use brew install'
    installDoc.linux: 'use apt install'

  web:
    cmd: ./node_modules/.bin/gulp --silent --color dev
    install: yarn install

  # [...]

commandsets:
  # This is the set that will be used when `sg start` is run:
  default:
    - gitserver
    - query-runner
    - repo-updater
    - searcher
    - symbols
    - github-proxy
    - frontend
    - watch
    - caddy
    - web
    - syntect_server
    - zoekt-indexserver-0
    - zoekt-indexserver-1
    - zoekt-webserver-0
    - zoekt-webserver-1
  # Another set that can be run with `sg run-set monitoring`:
  monitoring:
    - jaeger
    - docsite
    - prometheus
    - grafana
    - postgres_exporter
  enterprise:
    - enterprise-gitserver
    - enterprise-query-runner
    - enterprise-repo-updater
    - enterprise-frontend
    - searcher
    - symbols
    - github-proxy
    - watch
    - caddy
    - web
    - syntect_server
    - zoekt-indexserver-0
    - zoekt-indexserver-1
    - zoekt-webserver-0
    - zoekt-webserver-1

tests:
  # These can be run with `sg test [name]`
  backend:
    cmd: go test ./...
  backend-integration:
    cmd: go test -long -base-url $BASE_URL -email $EMAIL -username $USERNAME -password $PASSWORD
    env:
      # These are defaults. They can be overwritten by setting the env vars when
      # running the command.
      BASE_URL: 'http://localhost:3080'
      EMAIL: 'joe@sourcegraph.com'
      PASSWORD: '12345'
  frontend:
    cmd: yarn run jest --testPathIgnorePatterns end-to-end regression integration storybook
  frontend-e2e:
    cmd: yarn run mocha ./client/web/src/end-to-end/end-to-end.test.ts
    env:
      TS_NODE_PROJECT: client/web/src/end-to-end/tsconfig.json
```

## TODOs

- [ ] Rename `install` in the config files to `build` because it's clearer
- [ ] Add the remaining processes from `<root>/dev/Procfile` to `<root>/sg.config.yaml`
- [ ] All of the [ideas](#ideas) below
  - [ ] Rebuild and restart a command (if it has `build` defined, see Configuration): `sg build gitserver`
  - [ ] Implement the `sg generate` command
  - [ ] Implement `sg edit site-config` and `sg edit external-services`
  - [ ] Implement `sg tail-log`
- [ ] Add built-in support for "download binary" so that the `caddy` command, for example, would be 3 lines instead of 20. That would allow us to get rid of the bash code.

## Hacking

When you want to hack on `sg` it's best to be in the `dev/sg` directory and run it from there:

```
cd dev/sg
go run . -config ../../sg.config.yaml start
```

The `-config` can be anything you want, of course.

## Principles

- `sg` should be fun to use.
- If you think "it would be cool if `sg` could do X": add it! Let's go :)
- `sg` should make Sourcegraph developers productive and happy.
- `sg` is not and should not be a build system.
- `sg` is not and should not be a container orchestrator.
- Try to fix [a lot of the problems in this RFC](https://docs.google.com/document/d/18hrRIN0pUBRwUFF7vkcVmstJccqWeHiecNF2t1GAZfU/edit) by encoding conventions in executable code.
- No bash. `sg` was built to get rid of all the bash scripts in `./dev/`. If you have a chance to build something into `sg` to avoid another bash script: do it. Try to keep shell scripts to easy-to-understand one liners if you must. Replicating something in Go code that could be done in 4 lines of bash is probably a good idea.
- Duplicated data is fine as long as it's dumb data. Copying some lines in `sg.config.yaml` to get something working is often (but not always) better than trying to be clever.

You can also watch [this video](https://drive.google.com/file/d/1DXjjf1YXr8Od8vG4R74Ko-soLOx_tXa6/view?usp=sharing) to get an overview of the original thinking that lead to `sg`.

## Inspiration

- [GitLab Developer Kit (GDK)](https://gitlab.com/gitlab-org/gitlab-development-kit)
- Stripe's `pay` command, [described here](https://buttondown.email/nelhage/archive/papers-i-love-gg/)
- [Stack Exchange Local Environment Setup](https://twitter.com/nick_craver/status/1375871107773956103?s=21) command

## Ideas

The following are ideas for what could/should be built into `sg`:

#### Generators

```bash
# Generate code, equivalent to current `./dev/generate.sh`
sg generate

# Generate only specific things
sg generate mocks ./internal/enterprise/batches
```

#### Edit configuration files

```bash
# Edit the site configuration
sg edit site-config # opens site-config in $EDITOR
# Edit external service configuration
sg edit external-services # opens external-services.json in $EDITOR
```

#### Tail logs

```bash
# Tail the SQL logs
sg tail-log sql
# Tail the http logs
sg tail-log http
# Tail all logs
sg tail-log all
```
