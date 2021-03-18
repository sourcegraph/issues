# Running Vagrant based tests locally

## Requirements

- [Vagrant](https://www.vagrantup.com/downloads)
- [vagrant-env](https://github.com/gosuri/vagrant-env)
- [gcloud credentials](https://cloud.google.com/sdk/gcloud/reference/auth/login)

## Running tests

Each machine listed in [servers.yaml](servers.yaml) corresponds to a different test type. To list the available Vagrant boxes, execute:

```shell
$ vagrant status
Current machine states:

sourcegraph-e2e             not created (google)
sourcegraph-qa-test         not created (google)
sourcegraph-code-intel-test not created (google)
sourcegraph-upgrade         not created (google)
$
```

Copy [this 1password entry](https://start.1password.com/open/i?a=HEDEDSLHPBFGRBTKAKJWE23XX4&v=dnrhbauihkhjs5ag6vszsme45a&i=mn37wmu5dzhll6qxcnpmutvlq4&h=team-sourcegraph.1password.com) into a `.env` file this directory. This is used to populate the environment variables for each test.

To run the tests, simply execute the following:

```shell
$ vagrant up <machine>
```

where `<machine>` is one of those listed above.

Once the tests are finished, you can destroy the machine by executing the following:

```shell
$ vagrant destroy -f <machine>
```

To login into the machine by executing:

```shell
$ vagrant ssh <machine>
```

**Note:** these tests rsync the sourcegraph directory to the machine in Google Cloud. Depending on your connection speed, this could take a while.

## Adding tests

All machines are defined in the [servers.yaml](servers.yaml) file, and have a number of configuarable options based on the requirements of your test.

All commands should be wrapped into a bach script, which is run as part of the `shell_commands` block. An example of a new test is provided below, pay special attention to the `name` and the corresponding directory where the `test.sh` is stored.

```yaml
- name: new-vagrant-test
  box: google/gce
  machine_type: 'custom-16-20480'
  project_id: sourcegraph-ci
  external_ip: false
  use_private_ip: true
  network: default
  username: buildkite
  ssh_key_path: '~/.ssh/id_rsa'
  shell_commands:
    - |
      cd /sourcegraph
      ./dev/ci/test/new-vagrant-test/test.sh
```

(c) Copyright Sourcegraph 2012-2023.