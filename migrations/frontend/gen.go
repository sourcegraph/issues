// Package migrations contains the migration scripts for the DB.
package migrations

//go:generate env GOBIN=$PWD/.bin GO111MODULE=on go install github.com/kevinburke/go-bindata/go-bindata
//go:generate $PWD/.bin/go-bindata -nometadata -pkg migrations -ignore README.md -ignore .*\.go .
//go:generate gofmt -s -w bindata.go
