package main

import (
	"io"
	"os"

	"github.com/cockroachdb/errors"
	"gopkg.in/yaml.v2"
)

func ParseConfigFile(name string) (*Config, error) {
	name = "/Users/dax/work/sourcegraph/dev/sg/sg.config.example.yaml"
	file, err := os.Open(name)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot open file %q", name)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, errors.Wrap(err, "reading configuration file")
	}

	var conf Config
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return nil, err
	}

	for name, cmd := range conf.Commands {
		cmd.Name = name
		conf.Commands[name] = cmd
	}

	for name, cmd := range conf.Tests {
		cmd.Name = name
		conf.Tests[name] = cmd
	}

	for name, check := range conf.Checks {
		check.Name = name
		conf.Checks[name] = check
	}

	for name, cmd := range conf.Generators {
		cmd.Name = name
		conf.Generators[name] = cmd
	}

	return &conf, nil
}

type Command struct {
	Name             string
	Cmd              string            `yaml:"cmd"`
	Install          string            `yaml:"install"`
	CheckBinary      string            `yaml:"checkBinary"`
	Env              map[string]string `yaml:"env"`
	Watch            []string          `yaml:"watch"`
	InstallDocDarwin string            `yaml:"installDoc.darwin"`
	InstallDocLinux  string            `yaml:"installDoc.linux"`
	IgnoreStdout     bool              `yaml:"ignoreStdout"`
	IgnoreStderr     bool              `yaml:"ignoreStderr"`
	DefaultArgs      string            `yaml:"defaultArgs"`
	Directory        string            `yaml:"dir"`

	// ATTENTION: If you add a new field here, be sure to also handle that
	// field in `Merge` (below).
}

func (c Command) Merge(other Command) Command {
	merged := c

	if other.Name != merged.Name && other.Name != "" {
		merged.Name = other.Name
	}
	if other.Cmd != merged.Cmd && other.Cmd != "" {
		merged.Cmd = other.Cmd
	}
	if other.Install != merged.Install && other.Install != "" {
		merged.Install = other.Install
	}
	if other.InstallDocDarwin != merged.InstallDocDarwin && other.InstallDocDarwin != "" {
		merged.InstallDocDarwin = other.InstallDocDarwin
	}
	if other.InstallDocLinux != merged.InstallDocLinux && other.InstallDocLinux != "" {
		merged.InstallDocLinux = other.InstallDocLinux
	}
	if other.IgnoreStdout != merged.IgnoreStdout && !merged.IgnoreStdout {
		merged.IgnoreStdout = other.IgnoreStdout
	}
	if other.IgnoreStderr != merged.IgnoreStderr && !merged.IgnoreStderr {
		merged.IgnoreStderr = other.IgnoreStderr
	}
	if other.DefaultArgs != merged.DefaultArgs && other.DefaultArgs != "" {
		merged.DefaultArgs = other.DefaultArgs
	}

	if other.Directory != merged.Directory && other.Directory != "" {
		merged.Directory = other.Directory
	}

	for k, v := range other.Env {
		merged.Env[k] = v
	}

	if !equal(merged.Watch, other.Watch) {
		merged.Watch = other.Watch
	}

	return merged
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

type Check struct {
	Name        string `yaml:"-"`
	Cmd         string `yaml:"cmd"`
	FailMessage string `yaml:"failMessage"`
}

type Config struct {
	Env         map[string]string   `yaml:"env"`
	Commands    map[string]Command  `yaml:"commands"`
	Commandsets map[string][]string `yaml:"commandsets"`
	Tests       map[string]Command  `yaml:"tests"`
	Generators  map[string]Command  `yaml:"generators"`
	Checks      map[string]Check    `yaml:"checks"`
}

// Merges merges the top-level entries of two Config objects, with the receiver
// being modified.
func (c *Config) Merge(other *Config) {
	for k, v := range other.Env {
		c.Env[k] = v
	}

	for k, v := range other.Commands {
		if original, ok := c.Commands[k]; ok {
			c.Commands[k] = original.Merge(v)
		} else {
			c.Commands[k] = v
		}
	}

	for k, v := range other.Commandsets {
		c.Commandsets[k] = v
	}

	for k, v := range other.Tests {
		if original, ok := c.Tests[k]; ok {
			c.Tests[k] = original.Merge(v)
		} else {
			c.Tests[k] = v
		}
	}
	for k, v := range other.Generators {
		if original, ok := c.Generators[k]; ok {
			c.Generators[k] = original.Merge(v)
		} else {
			c.Generators[k] = v
		}
	}
}
