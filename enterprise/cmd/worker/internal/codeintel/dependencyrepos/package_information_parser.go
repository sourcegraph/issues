package dependencyrepos

import (
	"strings"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/lsifstore"
)

var schemeToParser map[string]func(lsifstore.Package) (DependencyRepoInfo, error) = map[string]func(lsifstore.Package) (DependencyRepoInfo, error){
	"semanticdb": parseJVMPackage,
}

func parseJVMPackage(packageReference lsifstore.Package) (DependencyRepoInfo, error) {
	replaced := strings.ReplaceAll(strings.TrimPrefix(packageReference.Name, "maven/"), "/", ":")
	return DependencyRepoInfo{
		Scheme:     packageReference.Scheme,
		Identifier: replaced,
		Version:    packageReference.Version,
	}, nil
}
