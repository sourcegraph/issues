package main

import (
	"bytes"
	"hash/fnv"
	"time"

	"github.com/fatih/color"

	"github.com/sourcegraph/batch-change-utils/output"
)

// tickDuration is the time to wait before writing the buffer contents
// without having received a newline.
var tickDuration time.Duration = 2 * time.Millisecond

// cmdLogger is a simplified version of goreman's logger:
// https://github.com/mattn/goreman/blob/master/log.go
type cmdLogger struct {
	out  *output.Output
	name string

	// buf is used to keep partial lines buffered before flushing them (either
	// on the next newline or after tickDuration)
	buf    *bytes.Buffer
	writes chan []byte
	done   chan struct{}
}

// newCmdLogger returns a new cmdLogger instance and spawns a goroutine in the
// background that regularily flushed the logged output to the given output..
func newCmdLogger(name string, out *output.Output) *cmdLogger {
	l := &cmdLogger{
		name:   name,
		out:    out,
		writes: make(chan []byte),
		done:   make(chan struct{}),
		buf:    &bytes.Buffer{},
	}

	go l.writeLines()

	return l
}

func (l *cmdLogger) bufLine(line []byte) error {
	_, err := l.buf.Write(line)
	return err
}

func (l *cmdLogger) flush() {
	if l.buf.Len() == 0 {
		return
	}
	// TODO: This always adds a newline, which is not always what we want. When
	// we flush partial lines, we don't want to add a newline character. What
	// we need to do: extend the `*output.Output` type to have a
	// `WritefNoNewline` (yes, bad name) method.
	l.out.Writef("%s[%s]%s %s", output.StyleBold, hashColor(l.name), output.StyleReset, l.buf.String())
	l.buf.Reset()
}

// Write handler of logger.
func (l *cmdLogger) Write(p []byte) (int, error) {
	l.writes <- p
	<-l.done
	return len(p), nil
}

func (l *cmdLogger) writeLines() {
	tick := time.NewTicker(tickDuration)
	for {
		select {
		case w, ok := <-l.writes:
			if !ok {
				l.flush()
				return
			}

			buf := bytes.NewBuffer(w)
			for {
				line, err := buf.ReadBytes('\n')
				if len(line) > 0 {
					if line[len(line)-1] == '\n' {
						// TODO: We currently add a newline in flush(), see comment there
						line = line[0 : len(line)-1]

						// But since there *was* a newline, we need to flush,
						// but only if there is more than a newline or there
						// was already content.
						if len(line) != 1 || l.buf.Len() > 0 {
							if err := l.bufLine(line); err != nil {
								break
							}
							l.flush()
						}
						tick.Stop()
					} else {
						if err := l.bufLine(line); err != nil {
							break
						}
						tick.Reset(tickDuration)
					}
				}
				if err != nil {
					break
				}
			}
			l.done <- struct{}{}
		case <-tick.C:
			l.flush()
			tick.Stop()
		}
	}
}

func hashColor(s string, v ...interface{}) string {
	colors := []func(string, ...interface{}) string{
		color.GreenString,
		color.CyanString,
		color.YellowString,
		color.BlueString,
		color.MagentaString,
		color.HiGreenString,
		color.HiCyanString,
		color.HiYellowString,
		color.HiBlueString,
		color.HiMagentaString,
	}
	h := fnv.New32()
	h.Write([]byte(s))
	colorFunc := colors[int(h.Sum32())%len(colors)]
	return colorFunc(s, v...)
}
