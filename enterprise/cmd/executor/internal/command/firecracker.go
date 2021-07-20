package command

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/lazyregexp"
)

type commandRunner interface {
	RunCommand(ctx context.Context, command command, logger *Logger) error
}

const firecrackerContainerDir = "/work"

var commonFirecrackerFlags = []string{
	"--runtime", "docker",
	"--network-plugin", "docker-bridge",
}

// formatFirecrackerCommand constructs the command to run on the host via a Firecracker
// virtual machine in order to invoke the given spec. If the spec specifies an image, then
// the command will be run inside of a container inside of the VM. Otherwise, the command
// will be run inside of the VM. The containers are one-shot and subject to the resource
// limits specified in the given options.
//
// The name value supplied here refers to the Firecracker virtual machine, which must have
// also been the name supplied to a successful invocation of setupFirecracker. Additionally,
// the virtual machine must not yet have been torn down (via teardownFirecracker).
func formatFirecrackerCommand(spec CommandSpec, name, repoDir string, options Options) command {
	rawOrDockerCommand := formatRawOrDockerCommand(spec, firecrackerContainerDir, options)

	innerCommand := strings.Join(rawOrDockerCommand.Command, " ")
	if len(rawOrDockerCommand.Env) > 0 {
		innerCommand = fmt.Sprintf("%s %s", strings.Join(rawOrDockerCommand.Env, " "), innerCommand)
	}
	if rawOrDockerCommand.Dir != "" {
		innerCommand = fmt.Sprintf("cd %s && %s", rawOrDockerCommand.Dir, innerCommand)
	}

	return command{
		Key:       spec.Key,
		Command:   []string{"ignite", "exec", name, "--", innerCommand},
		Operation: spec.Operation,
	}
}

// We've recently seen issues with concurent VM creation. It's likely we
// can do better here and run an empty VM at application startup, but I
// want to do this quick and dirty to see if we can raise our concurrency
// without other issues.
//
// https://github.com/weaveworks/ignite/issues/559
// Following up in https://github.com/sourcegraph/sourcegraph/issues/21377.
var igniteRunLock sync.Mutex

// setupFirecracker invokes a set of commands to provision and prepare a Firecracker virtual
// machine instance. This is done in several steps:
//
//   - For each of the docker images supplied, issue a `docker pull` and a `docker save`
//     to ensure we have an up-to-date tar archive of the requested image on the host.
//     These can be shared between different jobs.
//   - Provision a Firecracker VM (via ignite) subject to the resource limits specified
//     in the given options, and copy the contents of the working directory as well as
//     the docker image tar archives.
//   - Inside of the Firecracker VM, run docker load over all of the copied tarfiles so
//     that we do not need to pull the images from inside the VM, which has an empty
//     docker cache and would require us to pull images on every job.
func setupFirecracker(ctx context.Context, runner commandRunner, logger *Logger, name, repoDir string, imageNames, scriptPaths []string, options Options, operations *Operations) error {
	imageMap := map[string]string{}
	for i, image := range imageNames {
		imageMap[fmt.Sprintf("image%d", i)] = image
	}

	imageKeys := make([]string, 0, len(imageMap))
	for k := range imageMap {
		imageKeys = append(imageKeys, k)
	}
	sort.Strings(imageKeys)

	// Pull and archive each image that isn't already archived on the host
	for _, key := range imageKeys {
		if _, err := os.Stat(tarfilePathOnHost(key, options)); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}

		pullCommand := command{
			Key:       fmt.Sprintf("setup.docker.pull.%s", key),
			Command:   flatten("docker", "pull", imageMap[key]),
			Operation: operations.SetupDockerPull,
		}
		if err := runner.RunCommand(ctx, pullCommand, logger); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to pull %s", imageMap[key]))
		}

		saveCommand := command{
			Key:       fmt.Sprintf("setup.docker.save.%s", key),
			Command:   flatten("docker", "save", "-o", tarfilePathOnHost(key, options), imageMap[key]),
			Operation: operations.SetupDockerSave,
		}
		if err := runner.RunCommand(ctx, saveCommand, logger); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to save %s", imageMap[key]))
		}
	}

	// Start the VM and wait for the SSH server to become available
	startCommand := command{
		Key: "setup.firecracker.start",
		Command: flatten(
			"ignite", "run",
			commonFirecrackerFlags,
			firecrackerResourceFlags(options.ResourceOptions),
			firecrackerCopyfileFlags(repoDir, imageKeys, options),
			"--ssh",
			"--name", name,
			sanitizeImage(options.FirecrackerOptions.Image),
		),
		Operation: operations.SetupFirecrackerStart,
	}
	igniteRunLock.Lock()
	err := errors.Wrap(runner.RunCommand(ctx, startCommand, logger), "failed to start firecracker vm")
	igniteRunLock.Unlock()
	if err != nil {
		return err
	}

	// Load images from tar files
	for _, key := range imageKeys {
		loadCommand := command{
			Key:       fmt.Sprintf("setup.docker.load.%s", key),
			Command:   flatten("ignite", "exec", name, "--", "docker", "load", "-i", tarfilePathInVM(key)),
			Operation: operations.SetupDockerLoad,
		}
		if err := runner.RunCommand(ctx, loadCommand, logger); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to load %s", imageMap[key]))
		}
	}

	// Remove tar files to give more working space
	for _, key := range imageKeys {
		rmCommand := command{
			Key: fmt.Sprintf("setup.rm.%s", key),
			Command: flatten(
				"ignite", "exec", name, "--",
				"rm", tarfilePathInVM(key),
			),
			Operation: operations.SetupRm,
		}
		if err := runner.RunCommand(ctx, rmCommand, logger); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to remove tarfile for %s", imageMap[key]))
		}
	}

	return nil
}

// teardownFirecracker issues a stop and a remove request for the Firecracker VM with
// the given name.
func teardownFirecracker(ctx context.Context, runner commandRunner, logger *Logger, name string, options Options, operations *Operations) error {
	stopCommand := command{
		Key:       "teardown.firecracker.stop",
		Command:   flatten("ignite", "stop", name),
		Operation: operations.TeardownFirecrackerStop,
	}
	if err := runner.RunCommand(ctx, stopCommand, logger); err != nil {
		log15.Warn("Failed to stop firecracker vm", "name", name, "err", err)
	}

	removeCommand := command{
		Key:       "teardown.firecracker.remove",
		Command:   flatten("ignite", "rm", "-f", name),
		Operation: operations.TeardownFirecrackerRemove,
	}
	if err := runner.RunCommand(ctx, removeCommand, logger); err != nil {
		log15.Warn("Failed to remove firecracker vm", "name", name, "err", err)
	}

	return nil
}

func firecrackerResourceFlags(options ResourceOptions) []string {
	return []string{
		"--cpus", strconv.Itoa(options.NumCPUs),
		"--memory", options.Memory,
		"--size", options.DiskSpace,
	}
}

func firecrackerCopyfileFlags(dir string, imageKeys []string, options Options) []string {
	copyfiles := make([]string, 0, len(imageKeys)+1)
	for _, imageKey := range imageKeys {
		copyfiles = append(copyfiles, fmt.Sprintf(
			"%s:%s",
			tarfilePathOnHost(imageKey, options),
			tarfilePathInVM(imageKey),
		))
	}

	if dir != "" {
		copyfiles = append(copyfiles, fmt.Sprintf("%s:%s", dir, firecrackerContainerDir))
	}
	sort.Strings(copyfiles)

	return intersperse("--copy-files", copyfiles)
}

func tarfilePathOnHost(key string, options Options) string {
	return filepath.Join(options.FirecrackerOptions.ImageArchivesPath, fmt.Sprintf("%s.tar", key))
}

func tarfilePathInVM(key string) string {
	return fmt.Sprintf("/%s.tar", key)
}

var imagePattern = lazyregexp.New(`([^:@]+)(?::([^@]+))?(?:@sha256:([a-z0-9]{64}))?`)

// sanitizeImage sanitizes the given docker image for use by ignite. The ignite utility
// has some issue parsing docker tags that include a sha256 hash, so we try to remove it
// from any of the image references before passing it to the ignite command.
func sanitizeImage(image string) string {
	if matches := imagePattern.FindStringSubmatch(image); len(matches) == 4 {
		if matches[2] == "" {
			return matches[1]
		}

		return fmt.Sprintf("%s:%s", matches[1], matches[2])
	}

	return image
}
