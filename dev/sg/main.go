package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/sourcegraph/batch-change-utils/output"
)

var (
	runFlagSet = flag.NewFlagSet("sg run", flag.ExitOnError)
	runCommand = &ffcli.Command{
		Name:       "run",
		ShortUsage: "sg run <command>",
		ShortHelp:  "Run the given command.",
		FlagSet:    runFlagSet,
		Exec:       runExec,
		UsageFunc:  runUsage,
	}

	runSetFlagSet = flag.NewFlagSet("sg run-set", flag.ExitOnError)
	runSetCommand = &ffcli.Command{
		Name:       "run-set",
		ShortUsage: "sg run-set <commandset>",
		ShortHelp:  "Run the given command set.",
		FlagSet:    runSetFlagSet,
		Exec:       runSetExec,
		UsageFunc:  runSetUsage,
	}

	startFlagSet = flag.NewFlagSet("sg start", flag.ExitOnError)
	startCommand = &ffcli.Command{
		Name:       "start",
		ShortUsage: "sg start>",
		ShortHelp:  "Runs the commandset with the name 'start'.",
		FlagSet:    startFlagSet,
		Exec:       startExec,
		UsageFunc:  startUsage,
	}

	testFlagSet = flag.NewFlagSet("sg test", flag.ExitOnError)
	testCommand = &ffcli.Command{
		Name:       "test",
		ShortUsage: "sg test <testsuite>",
		ShortHelp:  "Run the given test suite.",
		FlagSet:    testFlagSet,
		Exec:       testExec,
		UsageFunc:  testUsage,
	}
)

var (
	rootFlagSet         = flag.NewFlagSet("sg", flag.ExitOnError)
	configFlag          = rootFlagSet.String("config", "sg.config.yaml", "configuration file")
	overwriteConfigFlag = rootFlagSet.String("overwrite", "sg.config.overwrite.yaml", "configuration overwrites file that is gitignored and can be used to, for example, add credentials")
	conf                *Config

	rootCommand = &ffcli.Command{
		ShortUsage:  "sg [flags] <subcommand>",
		FlagSet:     rootFlagSet,
		Subcommands: []*ffcli.Command{runCommand, runSetCommand, startCommand, testCommand},
	}
)

func main() {
	if err := rootCommand.Parse(os.Args[1:]); err != nil {
		fmt.Println(rootCommand.ShortUsage)
		os.Exit(1)
	}

	var err error
	conf, err = ParseConfigFile(*configFlag)
	if err != nil {
		out.WriteLine(output.Linef("", output.StyleWarning, "Failed to parse %s%s%s%s as configuration file:%s\n%s\n", output.StyleBold, *configFlag, output.StyleReset, output.StyleWarning, output.StyleReset, err))
		os.Exit(1)
	}

	if ok, _ := fileExists(*overwriteConfigFlag); ok {
		overwriteConf, err := ParseConfigFile(*overwriteConfigFlag)
		if err != nil {
			out.WriteLine(output.Linef("", output.StyleWarning, "Failed to parse %s%s%s%s as overwrites configuration file:%s\n%s\n", output.StyleBold, *overwriteConfigFlag, output.StyleReset, output.StyleWarning, output.StyleReset, err))
			os.Exit(1)
		}
		conf.Merge(overwriteConf)
	}

	if err := rootCommand.Run(context.Background()); err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
}

func runSetExec(ctx context.Context, args []string) error {
	if len(args) == 0 {
		out.WriteLine(output.Linef("", output.StyleWarning, "No commandset specified\n"))
		return flag.ErrHelp
	}

	if len(args) != 1 {
		out.WriteLine(output.Linef("", output.StyleWarning, "ERROR: too many arguments\n"))
		return flag.ErrHelp
	}

	names, ok := conf.Commandsets[args[0]]
	if !ok {
		out.WriteLine(output.Linef("", output.StyleWarning, "ERROR: commandset %q not found :(\n", args[0]))
		return flag.ErrHelp
	}

	cmds := make([]Command, 0, len(names))
	for _, name := range names {
		cmd, ok := conf.Commands[name]
		if !ok {
			return fmt.Errorf("command %q not found in commandset %q", name, args[0])
		}

		cmds = append(cmds, cmd)
	}

	return run(ctx, cmds...)
}

func testExec(ctx context.Context, args []string) error {
	if len(args) == 0 {
		out.WriteLine(output.Linef("", output.StyleWarning, "No test suite specified\n"))
		return flag.ErrHelp
	}

	if len(args) != 1 {
		out.WriteLine(output.Linef("", output.StyleWarning, "ERROR: too many arguments\n"))
		return flag.ErrHelp
	}

	cmd, ok := conf.Tests[args[0]]
	if !ok {
		out.WriteLine(output.Linef("", output.StyleWarning, "ERROR: test suite %q not found :(\n", args[0]))
		return flag.ErrHelp
	}

	return runTest(ctx, cmd)
}

func startExec(ctx context.Context, args []string) error {
	if len(args) != 0 {
		fmt.Printf("ERROR: this command doesn't take arguments\n\n")
		return flag.ErrHelp
	}

	return runSetExec(ctx, []string{"default"})
}

func runExec(ctx context.Context, args []string) error {
	if len(args) == 0 {
		out.WriteLine(output.Linef("", output.StyleWarning, "No command specified\n"))
		return flag.ErrHelp
	}

	if len(args) != 1 {
		out.WriteLine(output.Linef("", output.StyleWarning, "ERROR: too many arguments\n"))
		return flag.ErrHelp
	}

	cmd, ok := conf.Commands[args[0]]
	if !ok {
		out.WriteLine(output.Linef("", output.StyleWarning, "ERROR: command %q not found :(\n", args[0]))
		return flag.ErrHelp
	}

	return run(ctx, cmd)
}

func runUsage(c *ffcli.Command) string {
	var out strings.Builder

	fmt.Fprintf(&out, "USAGE\n")
	fmt.Fprintf(&out, "  sg %s <command>\n", c.Name)
	fmt.Fprintf(&out, "\n")
	fmt.Fprintf(&out, "AVAILABLE COMMANDS IN %s%s%s\n", output.StyleBold, *configFlag, output.StyleReset)

	for name := range conf.Commands {
		fmt.Fprintf(&out, "  %s\n", name)
	}

	return out.String()
}

func testUsage(c *ffcli.Command) string {
	var out strings.Builder

	fmt.Fprintf(&out, "USAGE\n")
	fmt.Fprintf(&out, "  sg %s <test suite>\n", c.Name)
	fmt.Fprintf(&out, "\n")
	fmt.Fprintf(&out, "AVAILABLE TESTSUITES IN %s%s%s\n", output.StyleBold, *configFlag, output.StyleReset)

	for name := range conf.Tests {
		fmt.Fprintf(&out, "  %s\n", name)
	}

	return out.String()
}

func runSetUsage(c *ffcli.Command) string {
	var out strings.Builder

	fmt.Fprintf(&out, "USAGE\n")
	fmt.Fprintf(&out, "  sg %s <commandset>\n", c.Name)
	fmt.Fprintf(&out, "\n")
	fmt.Fprintf(&out, "AVAILABLE COMMANDSETS IN %s%s%s\n", output.StyleBold, *configFlag, output.StyleReset)

	for name := range conf.Commandsets {
		fmt.Fprintf(&out, "  %s\n", name)
	}

	return out.String()
}

func startUsage(c *ffcli.Command) string {
	var out strings.Builder

	fmt.Fprintf(&out, "USAGE\n")
	fmt.Fprintln(&out, "  sg start")

	return out.String()
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
