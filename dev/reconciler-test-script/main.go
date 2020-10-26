package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db/basestore"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
)

// This is a test script that I use to manually integration-test the new
// changeset reconciler.
//
// It's pretty self-explanatory, I guess, and I want to keep it around for a bit.

const (
	// It's local access token, don't worry.
	authHeader = "token 1b13a0a1217377aa9a43d7cc46782f24b648ab0c"

	graphqlEndpoint = "http://localhost:3082/.api/graphql" // CI:LOCALHOST_OK
)

var deleteFlag = flag.Bool("del", false, "delete everything campaign-related in the DB before applying new campaign specs")

func main() {
	flag.Parse()
	if *deleteFlag {
		deleteEverything()
	}

	automationTestingID := getRepositoryID("github.com/sourcegraph/automation-testing")
	err := applySpecs(applyOpts{
		namespace:    "VXNlcjoxCg==", // User:1
		campaignSpec: newCampaignSpec("thorstens-campaign", "Updated description of my campaign"),
		changesetSpecs: []string{
			// `{"baseRepository":"` + automationTestingID + `","externalID":"1"}`,
			// `{"baseRepository":"` + automationTestingID + `","externalID":"311"}`,
			// `{"baseRepository":"` + automationTestingID + `","externalID":"309"}`,
			`{
			     "baseRepository": "` + automationTestingID + `",
			     "baseRev": "e4435274b43033cf0c212f61a2c16f7f2210cf56",
			     "baseRef":"refs/heads/master",

			     "headRepository": "` + automationTestingID + `",
			     "headRef":"refs/heads/retrying-changeset-creation",

			     "title": "The reconciler created this PR",
			     "body": "The reconciler also created this PR body",

			     "published": true,

			     "commits": [{
			    "message": "Pretty cool new commit message",
				"diff":` + fmt.Sprintf("%q", automationTestingDiff) + `}]
			}`,
		},
	})
	if err != nil {
		log.Fatalf("failed to apply specs: %s", err)
	}

	err = applySpecs(applyOpts{
		namespace:    "VXNlcjoxCg==",
		campaignSpec: newCampaignSpec("thorstens-2nd-campaign", "This is the second campaign"),
		changesetSpecs: []string{
			// `{"baseRepository":"` + automationTestingID + `","externalID":"311"}`,
			// `{"baseRepository":"` + automationTestingID + `","externalID":"309"}`,
			`{
			     "baseRepository": "` + automationTestingID + `",
			     "baseRev": "e4435274b43033cf0c212f61a2c16f7f2210cf56",
			     "baseRef":"refs/heads/master",

			     "headRepository": "` + automationTestingID + `",
			     "headRef":"refs/heads/thorstens-2nd-campaign",

			     "title": "PR in second campaign",
			     "body": "PR body in second campaign",

			     "published": true,

			     "commits": [{
			    "message": "Pretty commit message",
				"diff":` + fmt.Sprintf("%q", automationTestingDiff2) + `}]
			}`,
		},
	})
	if err != nil {
		log.Fatalf("failed to apply specs: %s", err)
	}
}

type applyOpts struct {
	namespace      string
	changesetSpecs []string
	campaignSpec   string
}

func applySpecs(opts applyOpts) error {
	var changesetSpecIDs []string

	for i, spec := range opts.changesetSpecs {
		fmt.Printf("Creating changesetSpec %d... ", i)

		q := fmt.Sprintf(createChangesetSpecTmpl, spec)
		res, err := sendRequest(q)
		if err != nil {
			return err
		}
		id := res.Data.CreateChangesetSpec.ID
		changesetSpecIDs = append(changesetSpecIDs, id)

		fmt.Printf("Done. ID: %s\n", id)
	}

	fmt.Printf("Creating campaignSpec... ")
	q := fmt.Sprintf(createCampaignSpecTmpl,
		opts.namespace,
		opts.campaignSpec,
		graphqlIDList(changesetSpecIDs...),
	)

	res, err := sendRequest(q)
	if err != nil {
		return fmt.Errorf("failed to create campaign spec: %s\n", err)
	}
	campaignSpecID := res.Data.CreateCampaignSpec.ID
	fmt.Printf("Done. ID: %s\n", campaignSpecID)

	fmt.Printf("Applying campaignSpec... ")
	q = fmt.Sprintf(applyCampaignTmpl, campaignSpecID)
	res, err = sendRequest(q)
	if err != nil {
		return fmt.Errorf("failed to apply campaign: %s\n", err)
	}
	campaignID := res.Data.ApplyCampaign.ID
	fmt.Printf("Done. Campaign ID: %s\n", campaignID)
	return nil
}

const applyCampaignTmpl = `
mutation ApplyCampaign { applyCampaign(campaignSpec: %q) { id } }
`

const createChangesetSpecTmpl = `
mutation CreateChangesetSpec {
  createChangesetSpec(changesetSpec: %q) {
    ... on HiddenChangesetSpec { id }
    ... on VisibleChangesetSpec { id }
  }
}
`

const createCampaignSpecTmpl = `
mutation CreateCampaignSpec {
  createCampaignSpec(namespace: %q, campaignSpec: %q, changesetSpecs: %s) {
    id
  }
}
`

type graphqlPayload struct {
	Query string
}

func graphqlIDList(ids ...string) string {
	var quoted []string
	for _, id := range ids {
		quoted = append(quoted, fmt.Sprintf("%q", id))
	}
	return fmt.Sprintf("[%s]", strings.Join(quoted, ", "))
}

type graphqlResponse struct {
	Data struct {
		CreateChangesetSpec struct {
			ID string
		}
		CreateCampaignSpec struct {
			ID string
		}
		ApplyCampaign struct {
			ID string
		}
	}
	Errors []struct {
		Message string
	}
}

func sendRequest(query string) (graphqlResponse, error) {
	var res graphqlResponse
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(graphqlPayload{Query: query})

	req, err := http.NewRequest("POST", graphqlEndpoint, b)
	if err != nil {
		return res, err
	}

	req.Header.Add("Authorization", authHeader)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return res, err
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return res, err
	}

	if len(res.Errors) != 0 {
		var messages []string
		for _, e := range res.Errors {
			messages = append(messages, e.Message)
		}
		list := strings.Join(messages, "\n- ")
		return res, fmt.Errorf("graphql errors:\n\t- %s\n", list)
	}

	return res, nil
}

func newCampaignSpec(name, description string) string {
	return fmt.Sprintf(`{
  "name": %q,
  "description": %q,
  "on": [
    {"repositoriesMatchingQuery": "lang:go func main"},
    {"repository": "github.com/sourcegraph/src-cli"}
  ],
  "steps": [
    {
      "run": "echo 'foobar'",
      "container": "alpine",
      "env": {
        "PATH": "/work/foobar:$PATH"
      }
    }
  ],
  "changesetTemplate": {
    "title": "Hello World",
    "body": "My first campaign!",
    "branch": "hello-world",
    "commit": {
      "message": "Append Hello World to all README.md files"
    },
    "published": false
  }
}`, name, description)
}

const automationTestingDiff = `diff --git test.md test.md
index 52ada66..0aaaf37 100644
--- test.md
+++ test.md
@@ -1 +1,3 @@
-# This is a test
+# This is a test.
+
+And this is another line
`

const automationTestingDiff2 = `diff --git test.md test.md
index 52ada66..0aaaf37 100644
--- test.md
+++ test.md
@@ -1 +1,3 @@
-# This is a test
+# What an amazing test!
+
+And this is another line
`

func deleteEverything() {
	ctx := context.Background()

	dsn := dbutil.PostgresDSN("", "sourcegraph", os.Getenv)
	db, err := dbutil.NewDB(dsn, "campaigns-reconciler")
	if err != nil {
		log.Fatalf("failed to initialize db store: %v", err)
	}

	if _, err := db.ExecContext(ctx, "DELETE FROM changeset_events;"); err != nil {
		log.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM changesets;"); err != nil {
		log.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM campaigns;"); err != nil {
		log.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM changeset_specs;"); err != nil {
		log.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM campaign_specs;"); err != nil {
		log.Fatal(err)
	}
}

func getRepositoryID(name string) string {
	dsn := dbutil.PostgresDSN("", "sourcegraph", os.Getenv)
	s, err := basestore.New(dsn, "campaigns-reconciler", sql.TxOptions{})
	if err != nil {
		log.Fatalf("failed to initialize db store: %v", err)
	}

	q := sqlf.Sprintf("select id from repo where name = %q", name)
	id, ok, err := basestore.ScanFirstInt(s.Query(context.Background(), q))
	if err != nil || !ok {
		log.Fatalf("querying repository id: %s", err)
	}
	return string(graphqlbackend.MarshalRepositoryID(api.RepoID(id)))
}

//
//            ____  __  ______________   _________    ____  ______
//           / __ \/ / / / ____/_  __/  /_  __/   |  / __ \/ ____/
//          / / / / / / / /     / /      / / / /| | / /_/ / __/
//         / /_/ / /_/ / /___  / /      / / / ___ |/ ____/ /___
//        /_____/\____/\____/ /_/      /_/ /_/  |_/_/   /_____/
//
//
//
//
//
//
//                                 .,/(##%##(((##
//          ,#((#(((((((((((((((((((((((((((((((((((#
//       (/////(((((//(((((((((((((////////////////////.
//     (///////(((/(//////////////////*//***/***********/
//    /**//*///(/((///(/*****************,,**,,**,**,**,,*/
//   ********/*///(//////*.,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,*
//  ***********///(////**** ................................,
// .*,**********//////****** ..........  ....................,
// *,,,,*****/*,*/////*******        .   .....................*
// ,,,,,,*,/.....,(#/******,,,  .. . .....................,.,.,,
// ,,,,,,,(....,,,,,,,***,*,,,, ................,..,,,,,,,,,,,,*
// ,,,,,,,..&*%.(%&@.,,,***,,** ...........,,,,,,,,,,,,,,,******/
// .,,,,,/..,%,.,/.(,,,,,**,****.,,,,,,,,,,,,,*,****************/
// ..,,,.*.  .... *#(,**,,******,*********************///////////(
// ..,,.,* .#.#.,,,,*,**/,******,,*******////////////////////////(
// ,...,,* *#&  /,,,,,***.,****,**////////////////////////////////
// ,..,,., /* % #.,,,&,***,**,,,**////////////////////////////////
//  ..,..., .,@....,,,,***,,,,,*,,//////////////////////////////*/.
//  ......,,    .#...,&,**,,,,***,///////////////*****************.
//   ......* *    %...,,,,,,,,,,,,********************************
//   ,......, #    &....,.,,,,,,,,************************////////
//    ....,,,* /   .. ...,,,,,,,,,***/////////////////////////////
//    ,.,,,.,,,, (   ...,,,,,,,,,/**//////////////////////////////
//     *,,,,,,,,,,,,,,,,,,,,,,,,,/,*******////////////////////////,
//      *,,,,,,,,,,,,,*,,,,,,,*,./#********/*/*////////////////////
//       ,,,,,,,,,,,,,*,*,*,,,,,/((,***************/**/**/**/******
//         ,,,,,,,,,,,,***,**,,*(((#*********************////////**/
//          *,,,,,,,,,,,,,**,**(((((.***********/*******************
//            .,,,,,,,,,,**,./((((###********************************
//               *,,,,,,*,,((###%&@&/********************************.
//                        ............********************************
//                        ............/******************************,*
//                          ...........******************************,,
//                            .........****,***,*******,,,,,,,,,,,,,,,,,
//                               .......*,,,,,*,,,,,,.,,,,,,,,,,,.,,,,,**/
//                                  .....,,,..............,,,,***////////////(.
//                                      ... .....,,,***//*/***********//////*
//                                         /*//****************//,
//                                            ./****//,
