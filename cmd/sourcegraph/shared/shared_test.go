package shared_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/sourcegraph/shared"
	"golang.org/x/net/context/ctxhttp"
)

// TODO(sqs): Manual steps are required:
//
// dropdb --if-exists cmd_sourcegraph_shared_test && createdb -O $USER cmd_sourcegraph_shared_test
func TestRun(t *testing.T) {
	os.Setenv("PGDATABASE", "cmd_sourcegraph_shared_test")
	log.SetFlags(0)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		if err := shared.RunAllSingleProcess(ctx); err != nil {
			t.Fatal(err)
		}
	}()

	time.Sleep(1000 * time.Millisecond)

	try := func(ctx context.Context) (retry bool, err error) {
		resp, err := ctxhttp.Get(ctx, nil, "http://localhost:7077")
		if err != nil {
			return true, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return false, fmt.Errorf("HTTP error response (%d)", resp.StatusCode)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}
		t.Log(string(body))
		return false, nil
	}
	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for {
		retry, err := try(ctx)
		if err == context.Canceled || err == context.DeadlineExceeded {
			t.Fatal(err)
		}
		if retry {
			time.Sleep(500 * time.Millisecond)
			t.Log("retry")
			continue
		}
		if err != nil {
			t.Fatal(err)
		}

		t.Log("ok")
		break
	}
}
