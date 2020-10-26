package indexer

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
)

type IndexJobLogger struct {
	m              sync.Mutex
	jobLogs        []*JobLog
	redactedValues []string
}

type JobLog struct {
	command []string
	out     *bytes.Buffer
}

func NewJobLogger(redactedValues ...string) *IndexJobLogger {
	return &IndexJobLogger{
		redactedValues: redactedValues,
	}
}

func (l *IndexJobLogger) RecordCommand(command []string, stdout, stderr io.Reader) {
	out := &bytes.Buffer{}

	l.m.Lock()
	l.jobLogs = append(l.jobLogs, &JobLog{command: command, out: out})
	l.m.Unlock()

	var m sync.Mutex
	var wg sync.WaitGroup

	readIntoBuf := func(prefix string, r io.Reader) {
		defer wg.Done()

		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			m.Lock()
			fmt.Fprintf(out, "%s: %s\n", prefix, scanner.Text())
			m.Unlock()
		}
	}

	wg.Add(2)
	go readIntoBuf("stdout", stdout)
	go readIntoBuf("stderr", stderr)
	wg.Wait()
}

func (l *IndexJobLogger) String() string {
	buf := &bytes.Buffer{}
	for _, jobLog := range l.jobLogs {
		payload := fmt.Sprintf("%s\n%s\n", strings.Join(jobLog.command, " "), jobLog.out)

		for _, v := range l.redactedValues {
			payload = strings.Replace(payload, v, "******", -1)
		}

		buf.WriteString(payload)
	}

	return buf.String()
}
