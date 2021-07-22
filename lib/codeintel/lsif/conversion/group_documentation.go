package conversion

import (
	"context"
	"strings"
	"sync"

	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/lib/codeintel/semantic"
)

// documentationChannels represents three result channels (pages, pathInfo, mappings), each being
// a queue (e.g. pages is the output stream, and enqueuePages is the input stream.)
//
// A goroutine for each queue sits in-between the channels and effectively ensures that writes to
// the enqueue channel do not block. Instead, they are queued up in a slice of dynamic memory. This
// is important because it means each of the channels of results can be fully consumed independently,
// i.e. you can read all pages before you read a single value from pathInfo/mappings, even though
// collectDocumentation works by incrementally building all three channels (and, without the queue,
// would fill up a channel and effectively become blocked.)
type documentationChannels struct {
	pages    chan *semantic.DocumentationPageData
	pathInfo chan *semantic.DocumentationPathInfoData
	mappings chan semantic.DocumentationMapping

	enqueuePages    chan *semantic.DocumentationPageData
	enqueuePathInfo chan *semantic.DocumentationPathInfoData
	enqueueMappings chan semantic.DocumentationMapping
}

func (c *documentationChannels) close() {
	close(c.enqueuePages)
	close(c.enqueuePathInfo)
	close(c.enqueueMappings)
}

func newDocumentationChannels() documentationChannels {
	channels := documentationChannels{
		pages:           make(chan *semantic.DocumentationPageData, 128),
		pathInfo:        make(chan *semantic.DocumentationPathInfoData, 128),
		mappings:        make(chan semantic.DocumentationMapping, 1024),
		enqueuePages:    make(chan *semantic.DocumentationPageData, 128),
		enqueuePathInfo: make(chan *semantic.DocumentationPathInfoData, 128),
		enqueueMappings: make(chan semantic.DocumentationMapping, 1024),
	}
	go func() {
		dst := channels.pages
		src := channels.enqueuePages
		var buf []*semantic.DocumentationPageData
		for {
			if len(buf) == 0 {
				v, ok := <-src
				if !ok {
					close(dst)
					return
				}
				buf = append(buf, v)
			}
			select {
			case dst <- buf[0]:
				buf = buf[1:]
			case v, ok := <-src:
				if !ok {
					close(dst)
					return
				}
				buf = append(buf, v)
			}
		}
	}()
	go func() {
		dst := channels.pathInfo
		src := channels.enqueuePathInfo
		var buf []*semantic.DocumentationPathInfoData
		for {
			if len(buf) == 0 {
				v, ok := <-src
				if !ok {
					close(dst)
					return
				}
				buf = append(buf, v)
			}
			select {
			case dst <- buf[0]:
				buf = buf[1:]
			case v, ok := <-src:
				if !ok {
					close(dst)
					return
				}
				buf = append(buf, v)
			}
		}
	}()
	go func() {
		dst := channels.mappings
		src := channels.enqueueMappings
		var buf []semantic.DocumentationMapping
		for {
			if len(buf) == 0 {
				v, ok := <-src
				if !ok {
					close(dst)
					return
				}
				buf = append(buf, v)
			}
			select {
			case dst <- buf[0]:
				buf = buf[1:]
			case v, ok := <-src:
				if !ok {
					close(dst)
					return
				}
				buf = append(buf, v)
			}
		}
	}()
	return channels
}

func collectDocumentation(ctx context.Context, state *State) documentationChannels {
	channels := newDocumentationChannels()
	if state.DocumentationResultRoot == -1 {
		channels.close()
		return channels
	}

	// Build a map of documentationResult IDs -> document IDs.
	documentationResultIDToDocumentID := map[int]int{}
	for documentID := range state.DocumentData {
		ranges := state.Contains.Get(documentID)
		if ranges != nil {
			ranges.Each(func(rangeID int) {
				rn := state.RangeData[rangeID]
				documentationResultIDToDocumentID[rn.DocumentationResultID] = documentID
			})
		}
	}

	pageCollector := &pageCollector{
		numWorkers:                  32,
		isChildPage:                 false,
		state:                       state,
		parentPathID:                "",
		startingDocumentationResult: state.DocumentationResultRoot,
		dupChecker:                  &duplicateChecker{pathIDs: make(map[string]struct{}, 16*1024)},
		walkedPages:                 &duplicateChecker{pathIDs: make(map[string]struct{}, 128)},
		lookupFilepath: func(documentationResultID int) *string {
			if documentID, ok := documentationResultIDToDocumentID[documentationResultID]; ok {
				tmp := state.DocumentData[documentID]
				return &tmp
			}
			return nil
		},
	}

	tmpPages := make(chan *semantic.DocumentationPageData)
	go pageCollector.collect(ctx, tmpPages, channels.enqueueMappings)
	go func() {
		// Emit path info for each page as a post-processing step once we've collected pages.
		for page := range tmpPages {
			var collectChildrenPages func(node *semantic.DocumentationNode) []string
			collectChildrenPages = func(node *semantic.DocumentationNode) []string {
				var children []string
				for _, child := range node.Children {
					if child.PathID != "" {
						children = append(children, child.PathID)
					} else if child.Node != nil {
						children = append(children, collectChildrenPages(child.Node)...)
					}
				}
				return children
			}
			isIndex := page.Tree.Label.Value == "" && page.Tree.Detail.Value == ""

			channels.enqueuePages <- page
			channels.enqueuePathInfo <- &semantic.DocumentationPathInfoData{
				PathID:   page.Tree.PathID,
				IsIndex:  isIndex,
				Children: collectChildrenPages(page.Tree),
			}
		}
		channels.close()
	}()
	return channels
}

type duplicateChecker struct {
	mu                        sync.RWMutex
	pathIDs                   map[string]struct{}
	duplicates, nonDuplicates int
}

func (d *duplicateChecker) add(pathID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.pathIDs[pathID]; ok {
		d.duplicates++
		return false
	}
	d.nonDuplicates++
	d.pathIDs[pathID] = struct{}{}
	return true
}

func (d *duplicateChecker) count() (duplicates, nonDupicates int) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.duplicates, d.nonDuplicates
}

// pageCollector collects all of the children for a single documentation page.
//
// It spawns a new pageCollector to collect each new child page as it encounters them.
type pageCollector struct {
	numWorkers                  int
	isChildPage                 bool
	parentPathID                string
	state                       *State
	startingDocumentationResult int
	dupChecker, walkedPages     *duplicateChecker
	lookupFilepath              func(documentationResultID int) *string
}

func (p *pageCollector) collect(ctx context.Context, ch chan<- *semantic.DocumentationPageData, mappings chan<- semantic.DocumentationMapping) (remainingPages []*pageCollector) {
	var walk func(parent *semantic.DocumentationNode, documentationResult int, pathID string)
	walk = func(parent *semantic.DocumentationNode, documentationResult int, pathID string) {
		labelID := p.state.DocumentationStringLabel[documentationResult]
		detailID := p.state.DocumentationStringDetail[documentationResult]
		documentation := p.state.DocumentationResultsData[documentationResult]
		this := &semantic.DocumentationNode{
			Documentation: documentation,
			Label:         p.state.DocumentationStringsData[labelID],
			Detail:        p.state.DocumentationStringsData[detailID],
		}
		switch {
		case pathID == "":
			this.PathID = "/"
		case this.Documentation.NewPage && pathID == "/":
			this.PathID = "/" + cleanPathIDElement(documentation.Identifier)
		case this.Documentation.NewPage:
			this.PathID = pathID + "/" + cleanPathIDElement(documentation.Identifier)
		default:
			this.PathID = pathID + "#" + cleanPathIDFragment(documentation.Identifier)
			if !p.dupChecker.add(this.PathID) {
				log15.Warn("API docs: duplicate pathID forbidden", "pathID", this.PathID)
				return
			}
		}
		if parent != nil {
			if this.Documentation.NewPage {
				// This documentationResult is a child of our parent, but it's a brand new page. We
				// spawn a new pageCollector to collect this page. We can't simply emit our page right
				// now, because we might not be finished collecting all the other descendant children
				// of this node.
				parent.Children = append(parent.Children, semantic.DocumentationNodeChild{
					PathID: this.PathID,
				})
				if p.walkedPages.add(this.PathID) {
					mappings <- semantic.DocumentationMapping{
						ResultID: uint64(documentationResult),
						PathID:   this.PathID,
						FilePath: p.lookupFilepath(documentationResult),
					}
					remainingPages = append(remainingPages, &pageCollector{
						isChildPage:                 true,
						parentPathID:                parent.PathID,
						state:                       p.state,
						startingDocumentationResult: documentationResult,
						dupChecker:                  p.dupChecker,
						walkedPages:                 p.walkedPages,
						lookupFilepath:              p.lookupFilepath,
					})
				}
				return
			} else {
				mappings <- semantic.DocumentationMapping{
					ResultID: uint64(documentationResult),
					PathID:   this.PathID,
					FilePath: p.lookupFilepath(documentationResult),
				}
				parent.Children = append(parent.Children, semantic.DocumentationNodeChild{
					Node: this,
				})
			}
		}

		children := p.state.DocumentationChildren[documentationResult]
		for _, child := range children {
			walk(this, child, pathIDTrimHash(this.PathID))
		}
		if documentationResult == p.startingDocumentationResult {
			// collected a whole page
			duplicates, nonDuplicates := p.dupChecker.count()
			if duplicates > 0 {
				log15.Error("API docs: upload failed due to duplicate pathIDs", "duplicates", duplicates, "nonDuplicates", nonDuplicates)
				return
			}
			ch <- &semantic.DocumentationPageData{Tree: this}
		}
	}
	walk(nil, p.startingDocumentationResult, p.parentPathID)
	if p.isChildPage {
		return remainingPages
	}

	// We are the root project page! Collect all the remaining pages.
	var (
		remainingWorkMu sync.RWMutex
		remainingWork   = remainingPages
	)
	wg := &sync.WaitGroup{}
	wg.Add(len(remainingWork))
	for i := 0; i <= p.numWorkers; i++ {
		go func() {
			for {
				// Get a remaining page to process.
				remainingWorkMu.Lock()
				if len(remainingWork) == 0 {
					remainingWorkMu.Unlock()
					return // no more work
				}
				work := remainingWork[0]
				remainingWork = remainingWork[1:]
				remainingWorkMu.Unlock()

				// Perform work.
				newRemainingPages := work.collect(ctx, ch, mappings)

				// Add new work, if needed.
				if len(newRemainingPages) > 0 {
					wg.Add(len(newRemainingPages))
					remainingWorkMu.Lock()
					remainingWork = append(remainingWork, newRemainingPages...)
					remainingWorkMu.Unlock()
				}
				wg.Done()
			}
		}()
	}
	wg.Wait()

	close(ch) // collected all pages
	return nil
}

// cleanPathIDElement replaces characters that may not be in URL path elements with dashes.
//
// It is not exhaustive, it only handles some common conflicts.
func cleanPathIDElement(s string) string {
	s = strings.Replace(s, "/", "-", -1)
	s = strings.Replace(s, "#", "-", -1)
	return s
}

// cleanPathIDFragment replaces characters that may not be in URL hashes with dashes.
//
// It is not exhaustive, it only handles some common conflicts.
func cleanPathIDFragment(s string) string {
	return strings.Replace(s, "#", "-", -1)
}

func joinPathIDs(a, b string) string {
	return a + "/" + b
}

func pathIDTrimHash(pathID string) string {
	i := strings.Index(pathID, "#")
	if i >= 0 {
		return pathID[:i]
	}
	return pathID
}
