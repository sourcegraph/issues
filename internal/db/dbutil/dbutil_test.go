package dbutil

import "testing"

func TestPostgresDSN(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		dsn  string
	}{{
		name: "default",
		env:  map[string]string{},
		dsn:  "postgres://testuser@127.0.0.1:5432",
	}, {
		name: "deploy-sourcegraph",
		env: map[string]string{
			"PGDATABASE": "sg",
			"PGHOST":     "pgsql",
			"PGPORT":     "5432",
			"PGSSLMODE":  "disable",
			"PGUSER":     "sg",
		},
		dsn: "postgres://sg@pgsql:5432/sg?sslmode=disable",
	}, {
		name: "deploy-sourcegraph password",
		env: map[string]string{
			"PGDATABASE": "sg",
			"PGHOST":     "pgsql",
			"PGPASSWORD": "REDACTED",
			"PGPORT":     "5432",
			"PGSSLMODE":  "disable",
			"PGUSER":     "sg",
		},
		dsn: "postgres://sg:REDACTED@pgsql:5432/sg?sslmode=disable",
	}, {
		name: "sourcegraph server",
		env: map[string]string{
			"PGHOST":     "127.0.0.1",
			"PGUSER":     "postgres",
			"PGDATABASE": "sourcegraph",
			"PGSSLMODE":  "disable",
		},
		dsn: "postgres://postgres@127.0.0.1/sourcegraph?sslmode=disable",
	}, {
		name: "datasource",
		env: map[string]string{
			"PGDATASOURCE": "postgres://foo@bar/bam",
		},
		dsn: "postgres://foo@bar/bam",
	}, {
		name: "datasource ignore",
		env: map[string]string{
			"PGHOST":       "pgsql",
			"PGDATASOURCE": "postgres://foo@bar/bam",
		},
		dsn: "postgres://foo@bar/bam",
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			have := PostgresDSN("", "testuser", func(e string) string {
				return tc.env[e]
			})
			if have != tc.dsn {
				t.Errorf("unexpected computed DSN\nhave: %s\nwant: %s", have, tc.dsn)
			}
		})
	}
}
