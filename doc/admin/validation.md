# Sourcegraph Instance Validation

>NOTE: **Sourcegraph Instance Validation is currently experimental.** We're exploring this feature set. 
>Let us know what you think! [File an issue](https://github.com/sourcegraph/sourcegraph/issues/new/choose)
>with feedback/problems/questions, or [contact us directly](https://about.sourcegraph.com/contact).

Instance validation provides a quick way to check that a Sourcegraph instance functions properly after a fresh install
 or an update.

The [`src` CLI](https://github.com/sourcegraph/src-cli) has an experimental command `validate` which drives the
 validation from a user-provided configuration file with a validation specification (in JSON or YAML format).

### Validation specification
 
The best way to describe this initial, simple and experimental validation specification is with the example below:

```yaml
# creates the first admin user on a fresh install (skips creation if user exists)
firstAdmin:
    email: foo@example.com
    username: foo
    password: "{{ .admin_password }}"

# adds the specified code host
externalService:
  config:
    url: https://github.com
    token: "{{ .github_token }}"
    orgs: []
    repos:
      - sourcegraph-testing/zap
  kind: GITHUB
  displayName: footest
  # set to true if this code host config should be deleted at the end of validation
  deleteWhenDone: true

# checks maxTries if specified repo is cloned and waits sleepBetweenTriesSeconds between checks 
waitRepoCloned:
  repo: github.com/footest/foo
  maxTries: 5
  sleepBetweenTriesSeconds: 2

# performs the specified search and checks that at least one result is returned
searchQuery: repo:^github.com/footest/foo$ uniquelyFoo
```

With this configuration, the validation command executes the following steps: 

* create the first admin user
* add an external service
* wait for a repository to be cloned
* perform a search
 
Every step is optional (if the corresponding top-level key is not present then the step is skipped).

### Passing in secrets

It is often the case that the config file with the validation specification needs to declare passwords, tokens or other
secrets and these secrets should not be exposed or committed to a git repo.

The validation specification can refer to string values that come from a context specified outside the config file
(see the `Usage` section below). References to string values from this outside context are specified like so:
`{{ .some_key }}`. The context will have a string value defined under the key `some_key` and the validation execution will
use that.

### Usage

Use the [`src` CLI](https://github.com/sourcegraph/src-cli) to validate:

```shell script
src validate -context github_token=$GITHUB_TOKEN validate.yaml
```

The `src` binary finds the Sourcegraph instance to validate from the environment variables 
[`SRC_ENDPOINT` and `SRC_ACCESS_TOKEN`](https://github.com/sourcegraph/src-cli#setup-with-your-sourcegraph-instance). 

> Note: The `SRC_ACCESS_TOKEN` is not needed when a first admin user is declared in the validation specification.
