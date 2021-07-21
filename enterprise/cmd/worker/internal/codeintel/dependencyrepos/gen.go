package dependencyrepos

//go:generate ../../../../../../dev/mockgen.sh github.com/sourcegraph/sourcegraph/enterprise/cmd/worker/internal/codeintel/dependencyrepos -i DBStore -i RepoUpdaterClient -o mock_iface_test.go
//go:generate ../../../../../../dev/mockgen.sh github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore -i PackageReferenceScanner -o mock_scanner_test.go
