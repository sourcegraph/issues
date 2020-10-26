package bitbucketserver

import (
	"testing"
)

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	cli, save := newClient(t, "BitbucketServer")
	defer save()

	f := newFixtures()
	f.load(t, cli)

	for _, tc := range []struct {
		name string
		test func(*testing.T)
	}{
		{"Provider/FetchAccount", testProviderFetchAccount(f, cli)},
		{"Provider/FetchUserPerms", testProviderFetchUserPerms(f, cli)},
		{"Provider/FetchRepoPerms", testProviderFetchRepoPerms(f, cli)},
	} {
		t.Run(tc.name, tc.test)
	}
}
