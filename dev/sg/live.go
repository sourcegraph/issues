package main

import (
	"io"
	"net/http"
	"strings"

	"github.com/cockroachdb/errors"
	"golang.org/x/mod/semver"

	"github.com/sourcegraph/sourcegraph/dev/sg/internal/command"
	"github.com/sourcegraph/sourcegraph/lib/output"
)

type environment struct {
	Name string
	URL  string
}

var environments = []environment{
	{Name: "dot-com", URL: "https://sourcegraph.com"},
	{Name: "k8s", URL: "https://k8s.sgdev.org"},
}

func environmentNames() []string {
	var names []string
	for _, e := range environments {
		names = append(names, e.Name)
	}
	return names
}

func getEnvironment(name string) (result environment, found bool) {
	for _, e := range environments {
		if e.Name == name {
			result = e
			found = true
		}
	}

	return result, found
}

func printDeployedVersion(e environment) error {
	pending := out.Pending(output.Linef("", output.StylePending, "Fetching newest version on %q...", e.Name))

	resp, err := http.Get(e.URL + "/__version")
	if err != nil {
		pending.Complete(output.Linef(output.EmojiFailure, output.StyleWarning, "Failed: %s", err))
		return err
	}
	defer resp.Body.Close()

	pending.Complete(output.Linef(output.EmojiSuccess, output.StyleSuccess, "Done"))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	bodyStr := string(body)
	if semver.IsValid("v" + bodyStr) {
		out.WriteLine(output.Linef(
			output.EmojiLightbulb, output.StyleLogo,
			"Live on %q: v%s",
			e.Name, bodyStr,
		))
		return nil
	}
	elems := strings.Split(bodyStr, "_")
	if len(elems) != 3 {
		return errors.Errorf("unknown format of /__version response: %q", body)
	}

	buildDate := elems[1]
	buildSha := elems[2]

	pending = out.Pending(output.Line("", output.StylePending, "Running 'git fetch' to update list of commits..."))
	_, err = command.RunGit("fetch", "-q")
	if err != nil {
		pending.Complete(output.Linef(output.EmojiFailure, output.StyleWarning, "Failed: %s", err))
		return err
	}
	pending.Complete(output.Linef(output.EmojiSuccess, output.StyleSuccess, "Done updating list of commits"))

	log, err := command.RunGit("log", "--oneline", "-n", "20", `--pretty=format:%h|%ar|%an|%s`, "origin/main")
	if err != nil {
		pending.Complete(output.Linef(output.EmojiFailure, output.StyleWarning, "Failed: %s", err))
		return err
	}

	out.Write("")
	line := output.Linef(
		output.EmojiLightbulb, output.StyleLogo,
		"Live on %q: %s%s%s %s(built on %s)",
		e.Name, output.StyleBold, buildSha, output.StyleReset, output.StyleLogo, buildDate,
	)
	out.WriteLine(line)

	out.Write("")

	var shaFound bool
	for _, logLine := range strings.Split(log, "\n") {
		elems := strings.SplitN(logLine, "|", 4)
		sha := elems[0]
		timestamp := elems[1]
		author := elems[2]
		message := elems[3]

		var emoji string = "  "
		var style output.Style = output.StylePending
		if sha[0:len(buildSha)] == buildSha {
			emoji = "🚀"
			style = output.StyleLogo
			shaFound = true
		}

		line := output.Linef(emoji, style, "%s (%s, %s): %s", sha, timestamp, author, message)
		out.WriteLine(line)
	}

	if !shaFound {
		line := output.Linef(output.EmojiWarning, output.StyleWarning, "Deployed SHA %s not found in last 20 commits on origin/main :(", buildSha)
		out.WriteLine(line)
	}

	return nil
}
