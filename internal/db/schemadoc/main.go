package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/lazyregexp"

	_ "github.com/lib/pq"
)

func runIgnoreError(cmd string, args ...string) {
	_ = exec.Command(cmd, args...).Run()
}

// This script generates markdown formatted output containing descriptions of
// the current dabase schema, obtained from postgres. The correct PGHOST,
// PGPORT, PGUSER etc. env variables must be set to run this script.
//
// First CLI argument is an optional filename to write the output to.
func generate(log *log.Logger) (string, error) {
	const dbname = "schemadoc-gen-temp"

	var (
		dataSource string
		run        func(cmd ...string) (string, error)
	)
	// If we are using pg9.6 use it locally since it is faster (CI \o/)
	versionRe := lazyregexp.New(fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta("9.6")))
	if out, _ := exec.Command("psql", "--version").CombinedOutput(); versionRe.Match(out) {
		dataSource = "dbname=" + dbname
		run = func(cmd ...string) (string, error) {
			c := exec.Command(cmd[0], cmd[1:]...)
			c.Stderr = log.Writer()
			out, err := c.Output()
			return string(out), err
		}
		runIgnoreError("dropdb", dbname)
		defer runIgnoreError("dropdb", dbname)
	} else {
		log.Printf("Running PostgreSQL 9.6 in docker since local version is %s", strings.TrimSpace(string(out)))
		if err := exec.Command("docker", "image", "inspect", "postgres:9.6").Run(); err != nil {
			log.Println("docker pull postgres9.6")
			pull := exec.Command("docker", "pull", "postgres:9.6")
			pull.Stdout = log.Writer()
			pull.Stderr = log.Writer()
			if err := pull.Run(); err != nil {
				return "", fmt.Errorf("docker pull postgres9.6 failed: %w", err)
			}
			log.Println("docker pull complete")
		}
		runIgnoreError("docker", "rm", "--force", dbname)
		server := exec.Command("docker", "run", "--rm", "--name", dbname, "-e", "POSTGRES_HOST_AUTH_METHOD=trust", "-p", "5433:5432", "postgres:9.6")
		if err := server.Start(); err != nil {
			return "", err
		}

		defer func() {
			_ = server.Process.Kill()
			runIgnoreError("docker", "kill", dbname)
			_ = server.Wait()
		}()

		dataSource = "postgres://postgres@127.0.0.1:5433/postgres?dbname=" + dbname
		run = func(cmd ...string) (string, error) {
			cmd = append([]string{"exec", "-u", "postgres", dbname}, cmd...)
			c := exec.Command("docker", cmd...)
			c.Stderr = log.Writer()
			out, err := c.Output()
			return string(out), err
		}

		attempts := 0
		for {
			attempts++
			if err := exec.Command("pg_isready", "-U", "postgres", "-d", dbname, "-h", "127.0.0.1", "-p", "5433").Run(); err == nil {
				break
			} else if attempts > 30 {
				return "", fmt.Errorf("gave up waiting after 30s attempt for pg_isready: %w", err)
			}
			time.Sleep(time.Second)
		}
	}

	if out, err := run("createdb", dbname); err != nil {
		return "", fmt.Errorf("createdb: %s: %w", out, err)
	}

	if err := dbconn.SetupGlobalConnection(dataSource); err != nil {
		return "", fmt.Errorf("SetupGlobalConnection: %w", err)
	}

	// Migrate the codeintel db on top of the frontend one so we capture
	// the schema of both databases.
	for _, databaseName := range dbutil.DatabaseNames {
		if err := dbconn.MigrateDB(dbconn.Global, databaseName); err != nil {
			return "", fmt.Errorf("MigrateDB: %w", err)
		}
	}

	db, err := dbconn.Open(dataSource)
	if err != nil {
		return "", fmt.Errorf("Open: %w", err)
	}

	// Query names of all public tables.
	rows, err := db.Query(`
SELECT table_name
FROM information_schema.tables
WHERE table_schema='public' AND table_type='BASE TABLE';
	`)
	if err != nil {
		return "", fmt.Errorf("Query: %w", err)
	}
	tables := []string{}
	defer rows.Close()
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		if err != nil {
			return "", fmt.Errorf("rows.Scan: %w", err)
		}
		tables = append(tables, name)
	}
	if err = rows.Err(); err != nil {
		return "", fmt.Errorf("rows.Err: %w", err)
	}

	docs := []string{}
	for _, table := range tables {
		// Get postgres "describe table" output.
		log.Println("describe", table)
		out, err := run("psql", "-X", "--quiet", "--dbname", dbname, "-c", fmt.Sprintf("\\d %s", table))
		if err != nil {
			return "", fmt.Errorf("describe %s failed: %w", table, err)
		}

		lines := strings.Split(string(out), "\n")
		doc := "# " + strings.TrimSpace(lines[0]) + "\n"
		doc += "```\n" + strings.Join(lines[1:], "\n") + "```\n"
		docs = append(docs, doc)
	}
	sort.Strings(docs)

	return strings.Join(docs, "\n"), nil
}

func main() {
	out, err := generate(log.New(os.Stderr, "", log.LstdFlags))
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 {
		if err := ioutil.WriteFile(os.Args[1], []byte(out), 0644); err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Print(out)
	}
}
