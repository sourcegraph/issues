module github.com/sourcegraph/sourcegraph/dev/sg

go 1.16

require (
	github.com/Masterminds/semver v1.5.0
	github.com/cockroachdb/errors v1.8.4
	github.com/golang-migrate/migrate/v4 v4.14.1
	github.com/google/go-cmp v0.5.5
	github.com/hashicorp/go-multierror v1.1.1
	github.com/jackc/pgx/v4 v4.11.0
	github.com/peterbourgon/ff/v3 v3.0.0
	github.com/rjeczalik/notify v0.9.2
	github.com/sourcegraph/sourcegraph/lib v0.0.0-00010101000000-000000000000
	golang.org/x/mod v0.4.2
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/sourcegraph/sourcegraph/lib v0.0.0-00010101000000-000000000000 => ./../../lib
