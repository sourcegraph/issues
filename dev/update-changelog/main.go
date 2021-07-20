package main

// update-changelog updates the provided markdown file, which is assumed to
// have a section of "unreleased changes", producing a new file which has
// those changes under a versioned header, and a new empty unreleased changes
// section.

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"

	"github.com/cockroachdb/errors"

	// "net"
	"os"
	"strings"
)

// a semantic representation of a given change set.

type changeSubset struct {
	heading string
	changes [][]byte
}

func (css *changeSubset) String() string {
	if css == nil {
		return ""
	}
	out := make([]string, 0, len(css.changes)+2)
	out = append(out, fmt.Sprintf("### %s\n", css.heading))
	out = append(out, "\n")
	for _, c := range css.changes {
		out = append(out, fmt.Sprintf("%s\n", c))
	}
	return strings.Join(out, "")
}

func (css *changeSubset) any() bool {
	return len(css.changes) > 0
}

type changeSet struct {
	version string
	subSets []*changeSubset
	verbose bool
}

func (cs *changeSet) String() string {
	if cs == nil {
		return ""
	}
	out := make([]string, 0, len(cs.subSets)+2)
	out = append(out, fmt.Sprintf("## %s\n", cs.version))
	out = append(out, "\n")
	for _, c := range cs.subSets {
		// We don't want to emit empty headings for existing
		// change sets, but we will for the new Unreleased
		// Changes section.
		if len(c.changes) > 0 || cs.verbose {
			out = append(out, fmt.Sprintf("%s\n", c))
		}
	}
	return strings.Join(out, "")
}

func (cs *changeSet) any() bool {
	for _, css := range cs.subSets {
		if css.any() {
			return true
		}
	}
	return false
}

// a representation of a changeLog in the format we use, which is ##
// headers per release, roughly.
type changeLog struct {
	header     [][]byte
	changeSets []*changeSet
}

var newVersion string

// readLines() yields the lines of the file as a slice of byte-slices
func readLines(path string) ([][]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// The arbitrary size reflects the knowledge that a changelog is probably
	// at least a couple hundred lines long.
	output := make([][]byte, 0, 512)
	lines := bufio.NewScanner(f)
	for lines.Scan() {
		output = append(output, []byte(lines.Text()))
	}
	return output, lines.Err()
}

// parseSubset tries to read a change subset, which is a ### header followed
// by a list of changes.
func parseSubset(sec [][]byte, lineCount int) (*changeSubset, error) {
	// the first subset can be a blank line, which is not an error
	if len(sec) < 1 || (len(sec) == 1 && len(sec[0]) == 0) {
		return nil, nil
	}
	if len(sec) < 2 {
		return nil, errors.Errorf("subsection not long enough")
	}
	if len(sec[0]) < 5 {
		return nil, errors.Errorf("subsection first line ('%s') not long enough", sec[0])
	}
	css := &changeSubset{heading: string(sec[0][4:])}
	for _, l := range sec[1:] {
		if len(l) > 0 {
			css.changes = append(css.changes, l)
		}
	}
	return css, nil
}

func parseSet(sec [][]byte, lineCount int) (*changeSet, error) {
	if len(sec) < 1 {
		return nil, errors.New("section not long enough to have a header")
	}
	// first line of a section has to be '## name'
	cs := &changeSet{version: string(sec[0][3:])}
	lineCount++
	subsections := make([][][]byte, 0)
	subsection := make([][]byte, 0)
	for _, l := range sec[1:] {
		if bytes.HasPrefix(l, []byte("### ")) {
			subsections = append(subsections, subsection)
			subsection = make([][]byte, 0)
		}
		subsection = append(subsection, l)
	}
	subsections = append(subsections, subsection)
	// sometimes there's uncategorized changes
	if len(subsections[0]) > 1 {
		forgery := [][]byte{[]byte("### Uncategorized")}
		forgery = append(forgery, subsections[0]...)
		subsections[0] = forgery
	}
	for _, sub := range subsections {
		subSet, err := parseSubset(sub, lineCount)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing subsection starting at %d: %s\n", lineCount, err)
			os.Exit(1)
		}
		if subSet != nil {
			cs.subSets = append(cs.subSets, subSet)
		}
		lineCount += len(sub)
	}
	// if there's uncategorized changes, fold them into Changed if it exists,
	// or make them the new Changed heading.
	if cs.subSets[0].heading == "Uncategorized" {
		found := false
		for _, css := range cs.subSets[1:] {
			if css.heading == "Changed" {
				found = true
				css.changes = append(css.changes, cs.subSets[0].changes...)
				cs.subSets = cs.subSets[1:]
				break
			}
		}
		if !found {
			cs.subSets[0].heading = "Changed"
		}
	}
	return cs, nil
}

func main() {
	flag.StringVar(&newVersion, "version", "", "the version number")
	flag.Parse()
	args := flag.Args()
	if newVersion == "" || len(args) != 1 {
		fmt.Fprintf(os.Stderr, "usage: update-changelog -version x.y.z filename\n")
		os.Exit(1)
	}
	lines, err := readLines(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading '%s': %s\n", args[0], err)
		os.Exit(1)
	}

	var sections [][][]byte
	section := make([][]byte, 0)
	for _, l := range lines {
		if bytes.HasPrefix(l, []byte("## ")) {
			sections = append(sections, section)
			section = make([][]byte, 0)
		}
		section = append(section, l)
	}
	sections = append(sections, section)

	cl := changeLog{header: sections[0]}
	lineCount := len(sections[0])
	for _, sec := range sections[1:] {
		cs, err := parseSet(sec, lineCount)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing section starting at line %d: %s\n", lineCount+1, err)
			os.Exit(1)
		}
		if cs != nil {
			cl.changeSets = append(cl.changeSets, cs)
		}
		lineCount += len(sec)
	}
	// look for an Unreleased Changes section, to tag with the requested version.
	if len(cl.changeSets) > 0 && strings.EqualFold(cl.changeSets[0].version, "unreleased changes") {
		if !cl.changeSets[0].any() {
			fmt.Fprintf(os.Stderr, "the unreleased changes section appears to be empty.")
			os.Exit(1)
		}
		cl.changeSets[0].version = newVersion
		newSet := changeSet{
			verbose: true,
			version: "Unreleased Changes",
			subSets: []*changeSubset{
				{heading: "Changed"},
				{heading: "Added"},
				{heading: "Fixed"},
			},
		}
		cl.changeSets = append([]*changeSet{&newSet}, cl.changeSets...)
	} else {
		fmt.Fprintf(os.Stderr, "cannot find unreleased changes section")
		os.Exit(1)
	}
	for _, h := range cl.header {
		fmt.Printf("%s\n", h)
	}
	for _, cs := range cl.changeSets {
		fmt.Printf("%s", cs)
	}
}
