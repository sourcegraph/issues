package indexer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
)

type CommandFormatter interface {
	Setup(ctx context.Context, commander Commander, images []string) error
	Teardown(ctx context.Context, commander Commander) error
	FormatCommand(cmd *Cmd) []string
}

type Cmd struct {
	image string
	// TODO(efritz) - currently treated as arguments and doesn't
	// support the intended image/commands setup, where each command
	// would be equivalent to a line in a bash script. Need to figure
	// out the best way to supply it to the underlying shell.
	command []string
	wd      string
	env     map[string]string
}

func NewCmd(image string, command ...string) *Cmd {
	return &Cmd{
		image:   image,
		command: command,
		env:     map[string]string{},
	}
}

func (cmd *Cmd) SetWd(wd string) *Cmd {
	cmd.wd = wd
	return cmd
}

func (cmd *Cmd) AddEnv(key, value string) *Cmd {
	cmd.env[key] = value
	return cmd
}

type dockerCommandFormatter struct {
	repoDir string
	options HandlerOptions
}

var _ CommandFormatter = &dockerCommandFormatter{}

func NewDockerCommandFormatter(
	repoDir string,
	options HandlerOptions,
) CommandFormatter {
	return &dockerCommandFormatter{
		repoDir: repoDir,
		options: options,
	}
}

func (r *dockerCommandFormatter) Setup(ctx context.Context, commander Commander, images []string) error {
	return nil
}

func (r *dockerCommandFormatter) Teardown(ctx context.Context, commander Commander) error {
	return nil
}

func (r *dockerCommandFormatter) FormatCommand(cmd *Cmd) []string {
	return flatten(
		"docker", "run", "--rm",
		r.resourceFlags(),
		r.volumeFlags(),
		r.workingdirectoryFlags(cmd.wd),
		r.envFlags(cmd.env),
		cmd.image,
		cmd.command,
	)
}

func (r *dockerCommandFormatter) resourceFlags() []string {
	return []string{
		"--cpus", strconv.Itoa(r.options.FirecrackerNumCPUs),
		"--memory", r.options.FirecrackerMemory,
	}
}

func (r *dockerCommandFormatter) volumeFlags() []string {
	return []string{"-v", fmt.Sprintf("%s:/data", r.repoDir)}
}

func (r *dockerCommandFormatter) workingdirectoryFlags(wd string) []string {
	return []string{"-w", filepath.Join("/data", wd)}
}

func (r *dockerCommandFormatter) envFlags(env map[string]string) []string {
	var flattened []string
	for _, key := range orderedKeys(env) {
		flattened = append(flattened, fmt.Sprintf("%s=%s", key, env[key]))
	}

	return intersperse("-e", flattened)
}

type firecrackerCommandFormatter struct {
	name      string
	repoDir   string
	options   HandlerOptions
	formatter CommandFormatter
}

var _ CommandFormatter = &firecrackerCommandFormatter{}

const FirecrackerRepoDir = "/repo-dir"

func NewFirecrackerCommandFormatter(
	name string,
	repoDir string,
	options HandlerOptions,
) CommandFormatter {
	return &firecrackerCommandFormatter{
		name:    name,
		repoDir: repoDir,
		options: options,
		formatter: NewDockerCommandFormatter(
			FirecrackerRepoDir,
			options,
		),
	}
}

var commonFirecrackerFlags = []string{
	"--runtime", "docker",
	"--network-plugin", "docker-bridge",
}

func (r *firecrackerCommandFormatter) Setup(ctx context.Context, commander Commander, images []string) error {
	imageMap := map[string]string{}
	for i, image := range images {
		imageMap[fmt.Sprintf("image%d", i)] = image
	}

	for _, key := range orderedKeys(imageMap) {
		if _, err := os.Stat(r.tarfilePathOnHost(key)); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}

		if err := r.saveDockerImage(ctx, commander, key, imageMap[key]); err != nil {
			return err
		}
	}

	startCommand := flatten(
		"ignite", "run",
		commonFirecrackerFlags,
		r.resourceFlags(),
		r.copyfileFlags(imageMap),
		"--ssh",
		"--name", r.name,
		sanitizeImage(r.options.FirecrackerImage),
	)
	if err := commander.Run(ctx, startCommand...); err != nil {
		return errors.Wrap(err, "failed to start firecracker vm")
	}

	for _, key := range orderedKeys(imageMap) {
		loadCommand := flatten(
			"ignite", "exec", r.name, "--",
			"docker", "load",
			"-i", r.tarfilePathInVM(key),
		)
		if err := commander.Run(ctx, loadCommand...); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to load %s", imageMap[key]))
		}
	}

	// Remove tar files inside of vm to clear scratch space
	for _, key := range orderedKeys(imageMap) {
		rmCommand := flatten(
			"ignite", "exec", r.name, "--",
			"rm", r.tarfilePathInVM(key),
		)
		if err := commander.Run(ctx, rmCommand...); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to remove tarfile for %s", imageMap[key]))
		}
	}

	return nil
}

func (r *firecrackerCommandFormatter) Teardown(ctx context.Context, commander Commander) error {
	stopCommand := flatten(
		"ignite", "stop",
		commonFirecrackerFlags,
		r.name,
	)
	if err := commander.Run(ctx, stopCommand...); err != nil {
		log15.Warn("Failed to stop firecracker vm", "name", r.name, "err", err)
	}

	removeCommand := flatten(
		"ignite", "rm", "-f",
		commonFirecrackerFlags,
		r.name,
	)
	if err := commander.Run(ctx, removeCommand...); err != nil {
		log15.Warn("Failed to remove firecracker vm", "name", r.name, "err", err)
	}

	return nil
}

func (r *firecrackerCommandFormatter) FormatCommand(cmd *Cmd) []string {
	return flatten("ignite", "exec", r.name, "--", r.formatter.FormatCommand(cmd))
}

func (r *firecrackerCommandFormatter) saveDockerImage(ctx context.Context, commander Commander, key, image string) error {
	pullCommand := flatten(
		"docker", "pull",
		image,
	)
	if err := commander.Run(ctx, pullCommand...); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to pull %s", image))
	}

	saveCommand := flatten(
		"docker", "save",
		"-o", r.tarfilePathOnHost(key),
		image,
	)
	if err := commander.Run(ctx, saveCommand...); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to save %s", image))
	}

	return nil
}

func (r *firecrackerCommandFormatter) resourceFlags() []string {
	return []string{
		"--cpus", strconv.Itoa(r.options.FirecrackerNumCPUs),
		"--memory", r.options.FirecrackerMemory,
		"--size", r.options.FirecrackerDiskSpace,
	}
}

func (r *firecrackerCommandFormatter) copyfileFlags(images map[string]string) (copyfiles []string) {
	for _, key := range orderedKeys(images) {
		copyfiles = append(copyfiles, fmt.Sprintf(
			"%s:%s",
			r.tarfilePathOnHost(key),
			r.tarfilePathInVM(key),
		))
	}

	return intersperse("--copy-files", append(
		[]string{fmt.Sprintf("%s:%s", r.repoDir, FirecrackerRepoDir)},
		copyfiles...,
	))
}

func (r *firecrackerCommandFormatter) tarfilePathOnHost(key string) string {
	return filepath.Join(r.options.ImageArchivePath, fmt.Sprintf("%s.tar", key))
}

func (r *firecrackerCommandFormatter) tarfilePathInVM(key string) string {
	return fmt.Sprintf("/%s.tar", key)
}

func orderedKeys(m map[string]string) (keys []string) {
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func intersperse(flag string, vs []string) (interspersed []string) {
	for _, v := range vs {
		interspersed = append(interspersed, flag, v)
	}

	return interspersed
}
