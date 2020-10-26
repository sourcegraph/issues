package repos

import (
	"time"

	"github.com/sourcegraph/sourcegraph/internal/conf"
)

func ConfRepoListUpdateInterval() time.Duration {
	v := conf.Get().RepoListUpdateInterval
	if v == 0 { //  default to 1 minute
		v = 1
	}
	return time.Duration(v) * time.Minute
}

func ConfRepoConcurrentExternalServiceSyncers() int {
	v := conf.Get().RepoConcurrentExternalServiceSyncers
	if v <= 0 {
		return 3
	}
	return v
}

func ConfUserReposMaxPerUser() int {
	v := conf.Get().UserReposMaxPerUser
	if v == 0 {
		return 2000
	}
	return v
}

func ConfUserReposMaxPerSite() int {
	v := conf.Get().UserReposMaxPerSite
	if v == 0 {
		return 200000
	}
	return v
}
