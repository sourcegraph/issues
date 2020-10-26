package secret

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"testing"

	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Verbose() {
		log15.Root().SetHandler(log15.DiscardHandler())
	}
	dbtesting.DBNameSuffix = "secret"
	os.Exit(m.Run())
}

func TestScanner(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()

	dbtesting.SetupGlobalTestDB(t)
	defaultEncryptor = newAESGCMEncodedEncryptor(mustGenerateRandomAESKey(), nil)

	t.Run("base", func(t *testing.T) {
		message := "Able was I ere I saw Elba"
		esMessage := StringValue{S: &message}

		_, err := dbconn.Global.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS secret_scanner_test(name text, message text)`)
		if err != nil {
			t.Fatal(err)
		}

		_, err = dbconn.Global.ExecContext(ctx, `INSERT INTO secret_scanner_test(name,message) VALUES ($1,$2)`, t.Name(), esMessage)
		if err != nil {
			t.Fatal(err)
		}

		var gotName string
		var gotMessage string
		err = dbconn.Global.QueryRowContext(ctx, `SELECT name,message FROM secret_scanner_test`).
			Scan(&gotName, &StringValue{S: &gotMessage})
		if err != nil {
			t.Fatal(err)
		}

		if gotName != t.Name() {
			t.Fatalf("expected %q, got %q for name", t.Name(), gotName)
		}
		if gotMessage != message {
			t.Fatalf("expected %q, got %q", message, gotMessage)
		}
	})

	t.Run("null example", func(t *testing.T) {
		_, err := dbconn.Global.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS secret_null_test(name text, message text)`)
		if err != nil {
			t.Fatal(err)
		}

		_, err = dbconn.Global.ExecContext(ctx, `INSERT INTO secret_null_test(name, message) VALUES ($1, $2)`, t.Name(), NullStringValue{})
		if err != nil {
			t.Fatal(err)
		}

		var gotName string
		var gotMessage string
		esMessage := NullStringValue{S: &gotMessage}
		err = dbconn.Global.QueryRowContext(ctx, `SELECT name,message FROM secret_null_test`).
			Scan(&gotName, &esMessage)
		if err != nil {
			t.Fatal(err)
		}

		if gotName != t.Name() {
			t.Fatalf("expected %q, got %q for name", t.Name(), gotName)
		}
		if esMessage.Valid {
			t.Fatal("expected not valid, got valid")
		}
	})

	t.Run("JSON", func(t *testing.T) {
		type record struct {
			CloneURL StringValue
		}

		cloneURL := "git@github.com:foo/bar.git"
		marshaled, err := json.Marshal(record{
			CloneURL: StringValue{S: &cloneURL},
		})
		if err != nil {
			t.Fatal(err)
		}

		var r record
		err = json.Unmarshal(marshaled, &r)
		if err != nil {
			t.Fatal(err)
		}

		if r.CloneURL.S == nil || *r.CloneURL.S != cloneURL {
			t.Fatalf("CloneURL: want %q but got %v", cloneURL, r.CloneURL.S)
		}
	})
}
