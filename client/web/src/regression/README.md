# Regression tests

This regression test suite is intended only for testing Sourcegraph prior to release. A long time ago, we performed manual QA tests as an organization prior to performing releases.

Today, this automated test suite effectively does what those manual QA tests used to - and we are trying to reduce the scope of this test suite by having individual teams improve their own unit and regression tests so this test suite is minimal.

## Running the tests

Prerequisites:

- A running Sourcegraph instance to which you have admin access (you'll need to create an
  admin-level `site-admin:sudo` access token).
- Ensure [src](https://github.com/sourcegraph/src-cli) is in your PATH.
- The regression tests will create test users as a side-effect. These are cleaned up if the tests
  run to completion, but if the tests are aborted, the lingering users should be cleaned up manually
  (failure to do so is a security risk, as the test user passwords may not be secure). Test
  usernames all begin with the prefix `test-`.
- Sourcegraph builtin authentication must be enabled and Sourcegraph must be directly accessible
  from the host that runs the test script (e.g., additional auth proxies will break the tests). This
  requirement may be removed at a later date.
- Install [`direnv`](https://direnv.net) and create a `.envrc` file at the root of this repository.

Run the tests:

1. From the repository root directory, `cd web` and then `yarn` and `yarn generate`.
1. From the repository root directory, `cd client/web`.
1. Create a `.envrc` file using the information in the 1Password Shared vault (look for `test:regression envrc` note.), and run `direnv allow` after.
1. Run `yarn run test:regression`.
1. You should see a Chrome window pop up and the tests will play in that window. The initial run may
   take awhile, because test repositories need to be cloned.
1. some tests require additional manual verification of screenshots after the test completes.
   Screenshots files are deposited in the current directory and are named descriptively for what
   should be checked.

Example:

At least the following is necessary to run the search regression tests against
the a local running Sourcegraph Docker image:

- The following environment variables must be set:

```bash
export SOURCEGRAPH_BASE_URL=http://localhost:7080
export GITHUB_TOKEN=<your-github-token>
export SOURCEGRAPH_SUDO_TOKEN=<your-sourcegraph-instance-token>
export SOURCEGRAPH_SUDO_USER=sourcegraph
export TEST_USER_PASSWORD=sourcegraph
export INCLUDE_ADMIN_ONBOARDING=false
export LOG_STATUS_MESSAGES=false
export NO_CLEANUP=true
```

- Start the Docker image `IMAGE=sourcegraph/server:VERSION ./dev/run-server-image.sh`

- Then run `yarn mocha src/regression/search.test.ts`

Tips:

- Use [`direnv`](https://direnv.net) to set environment variables automatically when you `cd` into
  the Sourcegraph repository directory
- Jest runs all tests even if an error occurs in initialization, so when an error occurs, you often
  have to scroll up--the first error is often the real one.
- When debugging test failures, you can insert the line `await new Promise(() => {})` to halt execution. Also read the [Puppeteer debugging
  docs](https://github.com/GoogleChrome/puppeteer#debugging-tips)
- When debugging you can also insert delays like so: `await new Promise( resolve => setTimeout(resolve, 1000));`. The unit
  of time is millis, so in that example it would wait for 1 second. This is useful if you want to open up the browser
  debugging tools before it proceeds.
- The `SLOWMO` environment variable will slow down Puppeteer execution by the specified number of
  milliseconds. `HEADLESS` will cause Puppeteer to run in headless mode (no visible browser window).
- Tests can be flakey. For the search tests, at least the following are known to be flakey:
  - `Global search for a filename with a few results`
  - `Text search non-master branch, large repository, many results`

## Adding a test

Test files live in `web/src/regression` and are split into different files according to feature
area (e.g., `search.test.ts`, `onboarding.test.ts`). The `util` subdirectory provides utility
packages. Tests also make use of the utility packages in `shared/src/testing`.

Add your test case to the appropriate file in `web/src/regression` or create a new one if it doesn't
match any of the existing files.

Test best practices:

- **Learn by example**: Read through the existing tests before implementing your own.
- **Use the test libraries**: Look at the utility packages in the `util` subdirectory, as these can
  save a lot of time, both in writing the initial test implementation and time down the road spent
  debugging flakiness and fragility (a common plague of GUI end-to-end tests).
- **Short and simple**: Bias toward short, quick tests. If you wish to test a long or complex
  workflow, consider breaking the workflow into different Jest `test` statements.
- **Security**: Don't have your test do anything a site admin user should not do. Assume your test
  will be run against a publicly accessible instance of Sourcegraph with access to private data. Any
  test user created should have a password configured by the `TEST_USER_PASSWORD` environment
  variable.
- **Non-destructive**: Your test should be non-destructive. It should not delete or modify existing
  users, repositories, settings, or other data on the instance. Any data it requires, it should add
  and then delete at the end of the test.
- **Clean up**: Your test should clean up after itself (_unless_ the `NO_CLEANUP` environment
  variable is set, in which case a test does not have to clean up if it is more performant not to do
  so.)
- **No side-effect dependencies**: Your tests should not depend on side effects of any other tests
  or be dependent on the order in which tests are run. Tests should in theory be able to run in
  parallel (though in practice, they are run serially).
- **Test usernames**: Any users your test creates should have a username prefix `test-`. This makes
  test users easy to clean up manually if necessary.
- **Ask for help**: Regression tests are intended to be relatively easy and straightforward to write. If you find
  yourself spending a lot of time implementing simple behavior, file an issue and tag @beyang.
