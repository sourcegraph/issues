package assets

//go:generate env GOBIN=$PWD/.bin GO111MODULE=on go install github.com/kevinburke/go-bindata/go-bindata
//go:generate $PWD/.bin/go-bindata -nometadata -pkg assets -ignore .*\.go .
//go:generate gofmt -s -w bindata.go
