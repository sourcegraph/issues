package ci

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	bk "github.com/sourcegraph/sourcegraph/internal/buildkite"
)

var allDockerImages = []string{
	// Slow images first for faster CI
	"server",
	"frontend",
	"grafana",
	"prometheus",
	"ignite-ubuntu",

	"github-proxy",
	"gitserver",
	"query-runner",
	"repo-updater",
	"searcher",
	"symbols",
	"precise-code-intel-bundle-manager",
	"precise-code-intel-worker",
	"precise-code-intel-indexer",
	"precise-code-intel-indexer-vm",

	// Images under docker-images/
	"cadvisor",
	"indexed-searcher",
	"postgres-11.4",
	"redis-cache",
	"redis-store",
	"search-indexer",
	"syntax-highlighter",
	"jaeger-agent",
	"jaeger-all-in-one",
	"codeintel-db",
}

// Verifies the docs formatting and builds the `docsite` command.
func addDocs(pipeline *bk.Pipeline) {
	pipeline.AddStep(":memo:",
		bk.Cmd("./dev/ci/yarn-run.sh prettier-check"),
		bk.Cmd("./dev/check/docsite.sh"))
}

// Adds the static check test step.
func addCheck(pipeline *bk.Pipeline) {
	pipeline.AddStep(":white_check_mark:", bk.Cmd("./dev/check/all.sh"))
}

// Adds the lint test step.
func addLint(pipeline *bk.Pipeline) {
	// If we run all lints together it is our slow step (5m). So we split it
	// into two and try balance the runtime. yarn is a fixed cost so we always
	// pay it on a step. Aim for around 3m.
	//
	// Random sample of timings:
	//
	// - yarn 41s
	// - eslint 137s
	// - build-ts 60s
	// - prettier 29s
	// - stylelint 7s
	// - graphql-lint 1s
	pipeline.AddStep(":eslint:",
		bk.Cmd("dev/ci/yarn-run.sh build-ts all:eslint")) // eslint depends on build-ts
	pipeline.AddStep(":lipstick: :lint-roller: :stylelint: :graphql:",
		bk.Cmd("dev/ci/yarn-run.sh prettier-check all:stylelint graphql-lint all:tsgql"))
}

// Adds steps for the OSS and Enterprise web app builds. Runs the web app tests.
func addWebApp(pipeline *bk.Pipeline) {
	// Webapp build
	pipeline.AddStep(":webpack::globe_with_meridians:",
		bk.Cmd("dev/ci/yarn-build.sh client/web"),
		bk.Env("NODE_ENV", "production"),
		bk.Env("ENTERPRISE", "0"))

	// Webapp enterprise build
	pipeline.AddStep(":webpack::globe_with_meridians::moneybag:",
		bk.Cmd("dev/ci/yarn-build.sh client/web"),
		bk.Env("NODE_ENV", "production"),
		bk.Env("ENTERPRISE", "1"))

	// Webapp tests
	pipeline.AddStep(":jest::globe_with_meridians:",
		bk.Cmd("dev/ci/yarn-test.sh client/web"),
		bk.Cmd("bash <(curl -s https://codecov.io/bash) -c -F typescript -F unit"))
}

// Builds and tests the browser extension.
func addBrowserExt(pipeline *bk.Pipeline) {
	// Browser extension build
	pipeline.AddStep(":webpack::chrome:",
		bk.Cmd("dev/ci/yarn-build.sh client/browser"))

	// Browser extension tests
	pipeline.AddStep(":jest::chrome:",
		bk.Cmd("dev/ci/yarn-test.sh client/browser"),
		bk.Cmd("bash <(curl -s https://codecov.io/bash) -c -F typescript -F unit"))
}

// Adds the shared frontend tests (shared between the web app and browser extension).
func addSharedTests(c Config) func(pipeline *bk.Pipeline) {
	return func(pipeline *bk.Pipeline) {
		// Client integration tests
		pipeline.AddStep(":puppeteer::electric_plug:",
			bk.Env("PUPPETEER_SKIP_CHROMIUM_DOWNLOAD", ""),
			bk.Env("ENTERPRISE", "1"),
			bk.Env("PERCY_ON", "true"),
			bk.Cmd("COVERAGE_INSTRUMENT=true dev/ci/yarn-run.sh build-web"),
			bk.Cmd("yarn percy exec -- yarn run cover-integration"),
			bk.Cmd("yarn nyc report -r json"),
			bk.Cmd("bash <(curl -s https://codecov.io/bash) -c -F typescript -F integration"),
			bk.ArtifactPaths("./puppeteer/*.png"))

		// Storybook coverage
		pipeline.AddStep(":storybook::codecov:",
			bk.Env("PUPPETEER_SKIP_CHROMIUM_DOWNLOAD", ""),
			bk.Cmd("COVERAGE_INSTRUMENT=true dev/ci/yarn-run.sh build-storybook"),
			bk.Cmd("yarn run cover-storybook"),
			bk.Cmd("yarn nyc report -r json"),
			bk.Cmd("bash <(curl -s https://codecov.io/bash) -c -F typescript -F storybook"))

		// Upload storybook to Chromatic
		chromaticCommand := "yarn chromatic --exit-zero-on-changes --exit-once-uploaded"
		if !c.isPR() {
			chromaticCommand += " --auto-accept-changes"
		}
		pipeline.AddStep(":chromatic:",
			bk.AutomaticRetry(5),
			bk.Cmd("yarn --mutex network --frozen-lockfile --network-timeout 60000"),
			bk.Cmd("yarn gulp generate"),
			bk.Cmd(chromaticCommand))

		// Shared tests
		pipeline.AddStep(":jest:",
			bk.Cmd("dev/ci/yarn-test.sh client/shared"),
			bk.Cmd("bash <(curl -s https://codecov.io/bash) -c -F typescript -F unit"))
	}
}

// Adds PostgreSQL backcompat tests.
func addPostgresBackcompat(pipeline *bk.Pipeline) {
	// TODO: We do not test Postgres DB backcompat anymore.
}

// Adds the Go test step.
func addGoTests(pipeline *bk.Pipeline) {
	pipeline.AddStep(":go:",
		bk.Cmd("./dev/ci/go-test.sh"),
		bk.Cmd("bash <(curl -s https://codecov.io/bash) -c -F go"))
}

// Builds the OSS and Enterprise Go commands.
func addGoBuild(pipeline *bk.Pipeline) {
	pipeline.AddStep(":go:",
		bk.Cmd("./dev/ci/go-build.sh"),
	)
}

// Lints the Dockerfiles.
func addDockerfileLint(pipeline *bk.Pipeline) {
	pipeline.AddStep(":docker:",
		bk.Cmd("./dev/ci/docker-lint.sh"))
}

// Adds backend integration tests step.
func addBackendIntegrationTests(c Config) func(*bk.Pipeline) {
	return func(pipeline *bk.Pipeline) {
		if !c.isMasterDryRun && c.branch != "master" && c.branch != "main" {
			return
		}

		pipeline.AddStep(":chains:",
			bk.Cmd("pushd enterprise"),
			bk.Cmd("./cmd/server/pre-build.sh"),
			bk.Cmd("./cmd/server/build.sh"),
			bk.Cmd("popd"),
			bk.Cmd("./dev/ci/backend-integration.sh"),
			bk.Cmd(`docker image rm -f "$IMAGE"`),
		)
	}
}

func addBrowserExtensionE2ESteps(pipeline *bk.Pipeline) {
	// TODO run this on firefox as well when our add-on is unblocked
	browser := "chrome"
	// Run e2e tests
	pipeline.AddStep(fmt.Sprintf(":%s:", browser),
		bk.Env("PUPPETEER_SKIP_CHROMIUM_DOWNLOAD", ""),
		bk.Env("EXTENSION_PERMISSIONS_ALL_URLS", "true"),
		bk.Env("BROWSER", browser),
		bk.Env("LOG_BROWSER_CONSOLE", "true"),
		bk.Env("SOURCEGRAPH_BASE_URL", "https://sourcegraph.com"),
		bk.Cmd("yarn --frozen-lockfile --network-timeout 60000"),
		bk.Cmd("pushd client/browser"),
		bk.Cmd("yarn -s run build"),
		bk.Cmd("yarn -s mocha ./src/end-to-end/github.test.ts ./src/end-to-end/gitlab.test.ts"),
		bk.Cmd("popd"),
		bk.ArtifactPaths("./puppeteer/*.png"))
}

// Release the browser extension.
func addBrowserExtensionReleaseSteps(pipeline *bk.Pipeline) {
	addBrowserExtensionE2ESteps(pipeline)

	pipeline.AddWait()

	// Release to the Chrome Webstore
	pipeline.AddStep(":rocket::chrome:",
		bk.Cmd("yarn --frozen-lockfile --network-timeout 60000"),
		bk.Cmd("pushd client/browser"),
		bk.Cmd("yarn -s run build"),
		bk.Cmd("yarn release:chrome"),
		bk.Cmd("popd"))

	// TODO uncomment this when our add-on will be unblocked
	// Build and self sign the FF extension and upload it to ...
	// pipeline.AddStep(":rocket::firefox:",
	// 	bk.Cmd("yarn --frozen-lockfile --network-timeout 60000"),
	// 	bk.Cmd("pushd client/browser"),
	// 	bk.Cmd("yarn release:ff"),
	// 	bk.Cmd("popd"))

	// Release to npm
	pipeline.AddStep(":rocket::npm:",
		bk.Cmd("yarn --frozen-lockfile --network-timeout 60000"),
		bk.Cmd("pushd client/browser"),
		bk.Cmd("yarn -s run build"),
		bk.Cmd("yarn release:npm"),
		bk.Cmd("popd"))
}

// Adds a Buildkite pipeline "Wait".
func wait(pipeline *bk.Pipeline) {
	pipeline.AddWait()
}

func triggerE2E(c Config, commonEnv map[string]string) func(*bk.Pipeline) {
	// Run e2e tests on release, patch and main branches
	runE2E := c.releaseBranch || c.taggedRelease || c.isBextReleaseBranch || c.patch || c.branch == "main"

	var async bool
	if c.branch == "main" {
		async = true
	} else {
		async = false
	}

	env := copyEnv(
		"BUILDKITE_PULL_REQUEST",
		"BUILDKITE_PULL_REQUEST_BASE_BRANCH",
		"BUILDKITE_PULL_REQUEST_REPO",
	)
	env["COMMIT_SHA"] = commonEnv["COMMIT_SHA"]
	env["DATE"] = commonEnv["DATE"]
	env["VERSION"] = commonEnv["VERSION"]
	env["CI_DEBUG_PROFILE"] = commonEnv["CI_DEBUG_PROFILE"]

	return func(pipeline *bk.Pipeline) {
		if !runE2E {
			return
		}

		pipeline.AddTrigger(":chromium:",
			bk.Trigger("sourcegraph-e2e"),
			bk.Async(async),
			bk.Build(bk.BuildOptions{
				Message: os.Getenv("BUILDKITE_MESSAGE"),
				Commit:  c.commit,
				Branch:  c.branch,
				Env:     env,
			}),
		)
	}
}

func copyEnv(keys ...string) map[string]string {
	m := map[string]string{}
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			m[k] = v
		}
	}
	return m
}

// Build all relevant Docker images for Sourcegraph, given the current CI case (e.g., "tagged
// release", "release branch", "master branch", etc.)
func addDockerImages(c Config, final bool) func(*bk.Pipeline) {
	addDockerImage := func(c Config, app string, insiders bool) func(*bk.Pipeline) {
		if !final {
			return addCandidateDockerImage(c, app)
		}
		return addFinalDockerImage(c, app, insiders)
	}

	return func(pipeline *bk.Pipeline) {
		switch {
		case c.taggedRelease:
			for _, dockerImage := range allDockerImages {
				addDockerImage(c, dockerImage, false)(pipeline)
			}
		case c.releaseBranch:
			addDockerImage(c, "server", false)(pipeline)
		case c.isMasterDryRun: // replicates `master` build but does not deploy
			for _, dockerImage := range allDockerImages {
				addDockerImage(c, dockerImage, false)(pipeline)
			}
		case c.branch == "master" || c.branch == "main":
			for _, dockerImage := range allDockerImages {
				addDockerImage(c, dockerImage, true)(pipeline)
			}

		case strings.HasPrefix(c.branch, "docker-images-patch/"):
			addDockerImage(c, c.branch[20:], false)(pipeline)
		}
	}
}

// Build a candidate docker image that will re-tagged with the final
// tags once the e2e tests pass.
func addCandidateDockerImage(c Config, app string) func(*bk.Pipeline) {
	return func(pipeline *bk.Pipeline) {

		baseImage := "sourcegraph/" + strings.ReplaceAll(app, "/", "-")

		cmds := []bk.StepOpt{
			bk.Cmd(fmt.Sprintf(`echo "Building candidate %s image..."`, app)),
			bk.Env("DOCKER_BUILDKIT", "1"),
			bk.Env("IMAGE", baseImage+":"+c.version),
			bk.Env("VERSION", c.version),
			bk.Cmd("yes | gcloud auth configure-docker"),
		}

		if _, err := os.Stat(filepath.Join("docker-images", app)); err == nil {
			// Building Docker image located under $REPO_ROOT/docker-images/
			cmds = append(cmds, bk.Cmd(filepath.Join("docker-images", app, "build.sh")))
		} else {
			// Building Docker images located under 4REPO_ROOT/cmd/
			cmdDir := func() string {
				if _, err := os.Stat(filepath.Join("enterprise/cmd", app)); err != nil {
					fmt.Fprintf(os.Stderr, "github.com/sourcegraph/sourcegraph/enterprise/cmd/%s does not exist so building github.com/sourcegraph/sourcegraph/cmd/%s instead\n", app, app)
					return "cmd/" + app
				}
				return "enterprise/cmd/" + app
			}()
			preBuildScript := cmdDir + "/pre-build.sh"
			if _, err := os.Stat(preBuildScript); err == nil {
				cmds = append(cmds, bk.Cmd(preBuildScript))
			}
			cmds = append(cmds, bk.Cmd(cmdDir+"/build.sh"))
		}

		gcrImage := fmt.Sprintf("us.gcr.io/sourcegraph-dev/%s", strings.TrimPrefix(baseImage, "sourcegraph/"))
		tag := candidateImageTag(c)
		cmds = append(cmds,
			bk.Cmd(fmt.Sprintf("docker tag %s:%s %s:%s", baseImage, c.version, gcrImage, tag)),
			bk.Cmd(fmt.Sprintf("docker push %s:%s", gcrImage, tag)),
		)

		pipeline.AddStep(":docker: :construction:", cmds...)
	}
}

// Tag and push final Docker image for the service defined by `app`
// after the e2e tests pass.
func addFinalDockerImage(c Config, app string, insiders bool) func(*bk.Pipeline) {
	return func(pipeline *bk.Pipeline) {
		baseImage := "sourcegraph/" + strings.ReplaceAll(app, "/", "-")
		gcrImage := fmt.Sprintf("us.gcr.io/sourcegraph-dev/%s", strings.TrimPrefix(baseImage, "sourcegraph/"))
		dockerHubImage := fmt.Sprintf("index.docker.io/%s", baseImage)

		var images []string
		for _, image := range []string{dockerHubImage, gcrImage} {
			if app != "server" || c.taggedRelease || c.patch || c.patchNoTest {
				images = append(images, fmt.Sprintf("%s:%s", image, c.version))
			}

			if app == "server" && c.releaseBranch {
				images = append(images, fmt.Sprintf("%s:%s-insiders", image, c.branch))
			}

			if insiders {
				images = append(images, fmt.Sprintf("%s:insiders", image))
			}
		}

		candidateImage := fmt.Sprintf("%s:%s", gcrImage, candidateImageTag(c))
		cmd := fmt.Sprintf("./dev/ci/docker-publish.sh %s %s", candidateImage, strings.Join(images, " "))

		pipeline.AddStep(":docker: :white_check_mark:", bk.Cmd(cmd))
	}
}

func candidateImageTag(c Config) string {
	buildNumber := os.Getenv("BUILDKITE_BUILD_NUMBER")
	return fmt.Sprintf("%s_%s_candidate", c.commit, buildNumber)
}
