package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/cockroachdb/errors"
	"github.com/drexedam/gravatar"
	gim "github.com/ozankasikci/go-image-merge"
	"github.com/slack-go/slack"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func main() {
	since := flag.Duration("since", 24*time.Hour, "Report new changelog entries since this period")
	dry := flag.Bool("dry", false, "If true, print out the JSON payload that would be sent to the Slack API")
	channel := flag.String("channel", "progress-bot-test", "Slack channel to post message to")
	gcsBucket := flag.String("bucket", "sourcegraph-progress-bot-avatars", "GCS bucket to which generated group avatars are uploaded")

	flag.Parse()

	blame, err := parseGitBlame(os.Stdin)
	if err != nil {
		log.Fatalf("failed to parse output of git blame: %v", err)
	}

	from := time.Now().UTC().Add(-*since)
	changelog, err := parseChangelog(blame, func(c *Change) bool {
		return !c.GitCommit.Author.Time.Before(from)
	})
	if err != nil {
		log.Fatalf("failed to parse CHANGELOG: %v", err)
	}

	ctx := context.Background()
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("failed to initialize GCP storage client: %v", err)
	}

	slackClient := NewSlackClient(slack.New(os.Getenv("SLACK_API_TOKEN")))

	bucket := gcsClient.Bucket(*gcsBucket)

	msg, err := changelog.ToSlackMessage(slackClient, bucket)
	if err != nil {
		log.Printf("Failed to generate Slack message: %v", err)
		os.Exit(0)
	}

	if *dry {
		json.NewEncoder(os.Stdout).Encode(msg)
		return
	}

	_, _, err = slackClient.PostMessage(
		*channel,
		slack.MsgOptionBlocks(msg.Blocks.BlockSet...),
		slack.MsgOptionIconEmoji(":rockyeet:"),
	)
	if err != nil {
		log.Fatalf("failed to post message to #%s: %v", *channel, err)
	}

	fmt.Printf("Posted message to #%s\n", *channel)
}

type Change struct {
	Release     string
	Description string
	Links       map[string]*Link
	GitCommit   *GitBlameLine
}

func (c Change) SlackText(userID string) string {
	description := c.Description
	for _, link := range c.Links {
		description = strings.ReplaceAll(description, link.Text, fmt.Sprintf("<%s|%s>", link.URL, link.Text))
	}

	if userID != "" {
		userID = "<@" + userID + ">"
	} else {
		userID = c.GitCommit.Author.Name
	}

	return fmt.Sprintf("• %s :writing_hand: %s", description, userID)
}

type Release struct {
	Release string   `json:"Release"`
	Added   []Change `json:"Added,omitempty"`
	Changed []Change `json:"Changed,omitempty"`
	Fixed   []Change `json:"Fixed,omitempty"`
	Removed []Change `json:"Removed,omitempty"`
}

func (r Release) IsEmpty() bool {
	return len(r.Added)+len(r.Changed)+len(r.Fixed)+len(r.Removed) == 0
}

type Changelog []Release

func (cl Changelog) ToSlackMessage(cli *SlackClient, bucket *storage.BucketHandle) (*slack.Message, error) {
	var merged Release
	for _, r := range cl {
		merged.Added = append(merged.Added, r.Added...)
		merged.Changed = append(merged.Changed, r.Changed...)
		merged.Fixed = append(merged.Fixed, r.Fixed...)
		merged.Removed = append(merged.Removed, r.Removed...)
	}

	section := func(name string, cs []Change) ([]slack.Block, error) {
		var text bytes.Buffer
		fmt.Fprintf(&text, "*%s*\n\n", name)

		avatarURLs := map[string]struct{}{}
		for _, c := range cs {
			var slackUserID string

			if strings.HasSuffix(c.GitCommit.Author.Email, "@sourcegraph.com") {
				user, err := cli.GetUserByEmail(c.GitCommit.Author.Email)
				if err != nil {
					log.Printf("slack.GetUserByEmail(%q): %v", c.GitCommit.Author.Email, err)
				} else {
					slackUserID = user.ID
					avatarURLs[user.Profile.Image48] = struct{}{}
				}
			} else {
				avatarURLs[gravatarURL(c.GitCommit.Author.Email)] = struct{}{}
			}

			fmt.Fprintln(&text, c.SlackText(slackUserID))
		}

		block := &slack.SectionBlock{
			Type: slack.MBTSection,
			Text: &slack.TextBlockObject{Type: "mrkdwn", Text: text.String()},
		}

		if len(avatarURLs) > 0 {
			imageURL, err := NewGroupAvatarImageURL(bucket, avatarURLs)
			if err != nil {
				log.Printf("Failed to generate group avatar: %v", err)
			} else {
				block.Accessory = slack.NewAccessory(slack.NewImageBlockElement(imageURL, "Group avatar"))
			}
		}

		return []slack.Block{
			slack.NewDividerBlock(),
			block,
		}, nil
	}

	blocks := []slack.Block{
		slack.NewHeaderBlock(&slack.TextBlockObject{
			Type: "plain_text",
			Text: "The CHANGELOG",
		}),
	}

	if merged.IsEmpty() {
		return nil, errors.Errorf("changelog is empty")
	}

	for _, s := range []struct {
		Name    string
		Changes []Change
	}{
		{"Added", merged.Added},
		{"Changed", merged.Changed},
		{"Fixed", merged.Fixed},
		{"Removed", merged.Removed},
	} {
		if len(s.Changes) > 0 {
			bs, err := section(s.Name, s.Changes)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, bs...)
		}
	}

	m := slack.NewBlockMessage(blocks...)

	return &m, nil
}

func gravatarURL(email string) string {
	return gravatar.New(email).
		Size(48).
		Default(gravatar.NotFound).
		Rating(gravatar.Pg).
		AvatarURL()
}

type SlackClient struct {
	*slack.Client

	mu    sync.RWMutex
	users map[string]*slack.User
}

func NewSlackClient(c *slack.Client) *SlackClient {
	return &SlackClient{
		Client: c,
		users:  make(map[string]*slack.User),
	}
}

func (c *SlackClient) GetUserByEmail(email string) (*slack.User, error) {
	c.mu.RLock()
	u, ok := c.users[email]
	c.mu.RUnlock()

	if ok {
		return u, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	u, err := c.Client.GetUserByEmail(email)
	if err != nil {
		return nil, err
	}

	c.users[email] = u

	return u, nil
}

func NewGroupAvatarImageURL(bucket *storage.BucketHandle, urls map[string]struct{}) (string, error) {
	sorted := make([]string, 0, len(urls))

	for url := range urls {
		sorted = append(sorted, url)
	}
	sort.Strings(sorted)

	grids := make([]*gim.Grid, len(sorted))

	var wg sync.WaitGroup
	wg.Add(len(sorted))

	for i, url := range sorted {
		i, url := i, url
		go func() {
			defer wg.Done()

			resp, err := http.Get(url)
			if err != nil {
				log.Printf("Failed to GET %q", url)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				log.Printf("Bad status code %v for %q", resp.StatusCode, url)
				return
			}

			avatar, _, err := image.Decode(resp.Body)
			if err != nil {
				log.Printf("Bad image from %v: %v", url, err)
				return
			}

			grids[i] = &gim.Grid{Image: &avatar, BackgroundColor: color.Transparent}
		}()
	}

	wg.Wait()

	filtered := grids[:0]
	for _, grid := range grids {
		if grid != nil {
			filtered = append(filtered, grid)
		}
	}

	if len(filtered) == 0 {
		return "", errors.Errorf("no avatar images")
	}

	merged, err := gim.New(filtered, 3, 3, func(m *gim.MergeImage) {
		m.BackgroundColor = color.Transparent
	}).Merge()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err = png.Encode(&buf, merged); err != nil {
		return "", err
	}

	digest := sha256.Sum256(buf.Bytes())

	ctx := context.Background()
	obj := bucket.Object(hex.EncodeToString(digest[:]))
	attrs, err := obj.Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		w := obj.If(storage.Conditions{DoesNotExist: true}).NewWriter(ctx)

		if _, err = io.Copy(w, &buf); err != nil {
			return "", err
		}

		if err = w.Close(); err != nil {
			return "", err
		}

		attrs = w.Attrs()
	}

	return attrs.MediaLink, nil
}

func parseChangelog(blame GitBlame, filter func(*Change) bool) (Changelog, error) {
	p := goldmark.DefaultParser()
	source := blame.Source()
	root := p.Parse(text.NewReader(source))

	var (
		changelog Changelog
		section   *[]Change
		release   *Release
	)

	err := ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n := n.(type) {
		case *ast.Heading:
			heading := string(bytes.TrimSpace(n.Text(source)))

			switch n.Level {
			case 2:
				if release != nil && !release.IsEmpty() {
					changelog = append(changelog, *release)
				}
				release = &Release{Release: heading}

				return ast.WalkContinue, nil
			case 3:
				switch heading {
				case "Added":
					section = &release.Added
				case "Changed":
					section = &release.Changed
				case "Fixed":
					section = &release.Fixed
				case "Removed":
					section = &release.Removed
				}

				return ast.WalkSkipChildren, nil
			}

			return ast.WalkContinue, nil

		case *ast.ListItem:
			if txt := n.FirstChild(); section != nil && txt != nil {
				ln := lineNumber(source, txt)
				if ln == -1 {
					return ast.WalkStop, errors.Errorf("found no blame line for %+v", n)
				}

				c := Change{
					GitCommit:   blame[ln],
					Description: string(bytes.TrimSpace(txt.Text(source))),
					Links:       findLinks(txt, source),
					Release:     release.Release,
				}

				if filter == nil || filter(&c) {
					*section = append(*section, c)
				}
			}
		}

		return ast.WalkContinue, nil
	})

	return changelog, err
}

type Link struct {
	URL  string
	Text string
}

func findLinks(n ast.Node, source []byte) (links map[string]*Link) {
	links = make(map[string]*Link)
	_ = ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch n := n.(type) {
		case *ast.Link:
			link := Link{
				URL:  string(n.Destination),
				Text: string(n.Text(source)),
			}
			links[link.Text] = &link
		}
		return ast.WalkContinue, nil
	})
	return
}

func lineNumber(source []byte, n ast.Node) int {
	lines := n.Lines()
	if lines == nil || lines.Len() == 0 {
		return -1
	}

	line := lines.At(0)
	return bytes.Count(source[:line.Start], []byte("\n"))
}

type GitBlame []*GitBlameLine

func (b GitBlame) Source() (source []byte) {
	for _, l := range b {
		source = append(source, l.Line...)
		source = append(source, '\n')
	}
	return
}

type GitSignature struct {
	Name  string
	Email string
	Time  time.Time
}

type GitBlameLine struct {
	Author    GitSignature
	Committer GitSignature

	Ref     string
	Message string

	Line string `json:"-"`
}

// git blame -w --line-porcelain
func parseGitBlame(r io.Reader) (b GitBlame, err error) {
	sc := bufio.NewScanner(r)

	var (
		l = new(GitBlameLine)
		n int
	)

	for sc.Scan() {
		line := sc.Text()
		switch n {
		case 0: // commit ID
			l.Ref = line[:strings.Index(line, " ")]
		case 1:
			l.Author.Name = strings.TrimPrefix(line, "author ")
		case 2:
			l.Author.Email = strings.Trim(strings.TrimPrefix(line, "author-mail "), "<>")
		case 3:
			ts, _ := strconv.ParseInt(strings.TrimPrefix(line, "author-time "), 10, 64)
			l.Author.Time = time.Unix(ts, 0).UTC()
		case 4:
			// ignore
		case 5:
			l.Committer.Name = strings.TrimPrefix(line, "committer ")
		case 6:
			l.Committer.Email = strings.Trim(strings.TrimPrefix(line, "committer-mail "), "<>")
		case 7:
			ts, _ := strconv.ParseInt(strings.TrimPrefix(line, "committer-time "), 10, 64)
			l.Committer.Time = time.Unix(ts, 0).UTC()
		case 8:
			// ignore
		case 9:
			l.Message = strings.TrimPrefix(line, "summary ")
		case 10, 11:
			// ignore
		case 12:
			l.Line = strings.TrimPrefix(line, "\t")
		}

		if n = (n + 1) % 13; n == 0 {
			b = append(b, l)
			l = new(GitBlameLine)
		}
	}

	return b, sc.Err()
}
