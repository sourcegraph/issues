package indexer

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	indexmanager "github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-indexer-vm/internal/index_manager"
	queuemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/queue/client/mocks"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
)

func init() {
	makeTempDir = func() (string, error) { return "/tmp/testing", nil }
}

func TestHandleWithDocker(t *testing.T) {
	queueClient := queuemocks.NewMockClient()
	indexManager := indexmanager.New()
	commander := NewMockCommander()

	handler := &Handler{
		queueClient:  queueClient,
		indexManager: indexManager,
		newCommander: func(*IndexJobLogger) Commander { return commander },
		options: HandlerOptions{
			FrontendURL:           "https://sourcegraph.test:1234",
			FrontendURLFromDocker: "https://sourcegraph.test:5432",
			AuthToken:             "hunter2",
			FirecrackerNumCPUs:    8,
			FirecrackerMemory:     "32G",
			FirecrackerDiskSpace:  "50G",
		},
		uuidGenerator: uuid.NewRandom,
	}

	index := store.Index{
		ID:             42,
		RepositoryName: "github.com/sourcegraph/sourcegraph",
		Commit:         "e2249f2173e8ca0c8c2541644847e7bf01aaef4a",
		DockerSteps: []store.DockerStep{
			{
				Root:     "r1",
				Image:    "install1",
				Commands: []string{"ls", "-liah"},
			}, {
				Root:     "r2",
				Image:    "install2",
				Commands: []string{"pwd"},
			},
		},
		Root:        "r3",
		Indexer:     "sourcegraph/lsif-go:latest",
		IndexerArgs: []string{"lsif-go", "--no-animation"},
		Outfile:     "nonstandard.lsif",
	}

	if err := handler.Handle(context.Background(), nil, index); err != nil {
		t.Fatalf("unexpected error handling index: %s", err)
	}

	if callCount := len(commander.RunFunc.History()); callCount != 7 {
		t.Errorf("unexpected run call count. want=%d have=%d", 7, callCount)
	} else {
		expectedCalls := []string{
			// Git commands
			"git -C /tmp/testing init",
			"git -C /tmp/testing -c protocol.version=2 fetch https://indexer:hunter2@sourcegraph.test:1234/.internal-code-intel/git/github.com/sourcegraph/sourcegraph e2249f2173e8ca0c8c2541644847e7bf01aaef4a",
			"git -C /tmp/testing checkout e2249f2173e8ca0c8c2541644847e7bf01aaef4a",
			// Docker steps
			"docker run --rm --cpus 8 --memory 32G -v /tmp/testing:/data -w /data/r1 install1 ls -liah",
			"docker run --rm --cpus 8 --memory 32G -v /tmp/testing:/data -w /data/r2 install2 pwd",
			// Index
			"docker run --rm --cpus 8 --memory 32G -v /tmp/testing:/data -w /data/r3 sourcegraph/lsif-go:latest lsif-go --no-animation",
			// Upload
			"docker run --rm --cpus 8 --memory 32G -v /tmp/testing:/data -w /data/r3 -e SRC_ENDPOINT=https://indexer:hunter2@sourcegraph.test:5432 sourcegraph/src-cli:latest lsif upload -no-progress -repo github.com/sourcegraph/sourcegraph -commit e2249f2173e8ca0c8c2541644847e7bf01aaef4a -upload-route /.internal-code-intel/lsif/upload -file nonstandard.lsif",
		}

		calls := commander.RunFunc.History()

		for i, expectedCall := range expectedCalls {
			if diff := cmp.Diff(expectedCall, strings.Join(calls[i].Arg1, " ")); diff != "" {
				t.Errorf("unexpected command (-want +got):\n%s", diff)
			}
		}
	}
}

func TestHandleWithFirecracker(t *testing.T) {
	queueClient := queuemocks.NewMockClient()
	indexManager := indexmanager.New()
	commander := NewMockCommander()

	handler := &Handler{
		queueClient:  queueClient,
		indexManager: indexManager,
		newCommander: func(*IndexJobLogger) Commander { return commander },
		options: HandlerOptions{
			FrontendURL:           "https://sourcegraph.test:1234",
			FrontendURLFromDocker: "https://sourcegraph.test:5432",
			AuthToken:             "hunter2",
			UseFirecracker:        true,
			FirecrackerImage:      "sourcegraph/ignite-ubuntu:latest",
			FirecrackerNumCPUs:    8,
			FirecrackerMemory:     "32G",
			FirecrackerDiskSpace:  "50G",
			ImageArchivePath:      "/images",
		},
		uuidGenerator: func() (uuid.UUID, error) {
			return uuid.MustParse("97b45daf-53d1-48ad-b992-547469d8e438"), nil
		},
	}

	index := store.Index{
		ID:             42,
		RepositoryName: "github.com/sourcegraph/sourcegraph",
		Commit:         "e2249f2173e8ca0c8c2541644847e7bf01aaef4a",
		DockerSteps: []store.DockerStep{
			{
				Root:     "r1",
				Image:    "install1",
				Commands: []string{"ls", "-liah"},
			}, {
				Root:     "r2",
				Image:    "install2",
				Commands: []string{"pwd"},
			},
		},
		Root:        "r3",
		Indexer:     "sourcegraph/lsif-go:latest",
		IndexerArgs: []string{"lsif-go", "--no-animation"},
		Outfile:     "nonstandard.lsif",
	}

	if err := handler.Handle(context.Background(), nil, index); err != nil {
		t.Fatalf("unexpected error handling index: %s", err)
	}

	if callCount := len(commander.RunFunc.History()); callCount != 26 {
		t.Errorf("unexpected run call count. want=%d have=%d", 26, callCount)
	} else {
		expectedCalls := []string{
			// Git commands
			"git -C /tmp/testing init",
			"git -C /tmp/testing -c protocol.version=2 fetch https://indexer:hunter2@sourcegraph.test:1234/.internal-code-intel/git/github.com/sourcegraph/sourcegraph e2249f2173e8ca0c8c2541644847e7bf01aaef4a",
			"git -C /tmp/testing checkout e2249f2173e8ca0c8c2541644847e7bf01aaef4a",
			// Stash docker images
			"docker pull sourcegraph/src-cli:latest",
			"docker save -o /images/image0.tar sourcegraph/src-cli:latest",
			"docker pull install1",
			"docker save -o /images/image1.tar install1",
			"docker pull install2",
			"docker save -o /images/image2.tar install2",
			"docker pull sourcegraph/lsif-go:latest",
			"docker save -o /images/image3.tar sourcegraph/lsif-go:latest",
			// VM setup
			"ignite run --runtime docker --network-plugin docker-bridge --cpus 8 --memory 32G --size 50G --copy-files /tmp/testing:/repo-dir --copy-files /images/image0.tar:/image0.tar --copy-files /images/image1.tar:/image1.tar --copy-files /images/image2.tar:/image2.tar --copy-files /images/image3.tar:/image3.tar --ssh --name 97b45daf-53d1-48ad-b992-547469d8e438 sourcegraph/ignite-ubuntu:latest",
			// Docker-inside-VM setup
			"ignite exec 97b45daf-53d1-48ad-b992-547469d8e438 -- docker load -i /image0.tar",
			"ignite exec 97b45daf-53d1-48ad-b992-547469d8e438 -- docker load -i /image1.tar",
			"ignite exec 97b45daf-53d1-48ad-b992-547469d8e438 -- docker load -i /image2.tar",
			"ignite exec 97b45daf-53d1-48ad-b992-547469d8e438 -- docker load -i /image3.tar",
			"ignite exec 97b45daf-53d1-48ad-b992-547469d8e438 -- rm /image0.tar",
			"ignite exec 97b45daf-53d1-48ad-b992-547469d8e438 -- rm /image1.tar",
			"ignite exec 97b45daf-53d1-48ad-b992-547469d8e438 -- rm /image2.tar",
			"ignite exec 97b45daf-53d1-48ad-b992-547469d8e438 -- rm /image3.tar",
			// Docker steps
			"ignite exec 97b45daf-53d1-48ad-b992-547469d8e438 -- docker run --rm --cpus 8 --memory 32G -v /repo-dir:/data -w /data/r1 install1 ls -liah",
			"ignite exec 97b45daf-53d1-48ad-b992-547469d8e438 -- docker run --rm --cpus 8 --memory 32G -v /repo-dir:/data -w /data/r2 install2 pwd",
			// Index
			"ignite exec 97b45daf-53d1-48ad-b992-547469d8e438 -- docker run --rm --cpus 8 --memory 32G -v /repo-dir:/data -w /data/r3 sourcegraph/lsif-go:latest lsif-go --no-animation",
			// Upload
			"ignite exec 97b45daf-53d1-48ad-b992-547469d8e438 -- docker run --rm --cpus 8 --memory 32G -v /repo-dir:/data -w /data/r3 -e SRC_ENDPOINT=https://indexer:hunter2@sourcegraph.test:5432 sourcegraph/src-cli:latest lsif upload -no-progress -repo github.com/sourcegraph/sourcegraph -commit e2249f2173e8ca0c8c2541644847e7bf01aaef4a -upload-route /.internal-code-intel/lsif/upload -file nonstandard.lsif",
			// Teardown
			"ignite stop --runtime docker --network-plugin docker-bridge 97b45daf-53d1-48ad-b992-547469d8e438",
			"ignite rm -f --runtime docker --network-plugin docker-bridge 97b45daf-53d1-48ad-b992-547469d8e438",
		}

		calls := commander.RunFunc.History()

		for i, expectedCall := range expectedCalls {
			if diff := cmp.Diff(expectedCall, strings.Join(calls[i].Arg1, " ")); diff != "" {
				t.Errorf("unexpected command (-want +got):\n%s", diff)
			}
		}
	}
}
