package authz

import (
	"testing"

	"github.com/RoaringBitmap/roaring"
	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
)

func TestPermsInclude(t *testing.T) {
	for _, tc := range []struct {
		Perms
		other Perms
		want  bool
	}{
		{None, Read, false},
		{None, Write, false},
		{Read, Read, true},
		{Read, None, true},
		{Read, Write, false},
		{Read, Read | Write, false},
		{Write, Write, true},
		{Write, Read, false},
		{Write, None, true},
		{Write, Read | Write, false},
		{Read | Write, Read, true},
		{Read | Write, Write, true},
		{Read | Write, None, true},
		{Read | Write, Write | Read, true},
	} {
		if have, want := tc.Include(tc.other), tc.want; have != want {
			t.Logf("%032b", tc.Perms&tc.other)
			t.Errorf(
				"\nPerms{%032b} Include\nPerms{%032b}\nhave: %t\nwant: %t",
				tc.Perms,
				tc.other,
				have, want,
			)
		}
	}
}

func BenchmarkPermsInclude(b *testing.B) {
	p := Read | Write
	for i := 0; i < b.N; i++ {
		_ = p.Include(Write)
	}
}

func TestPermsString(t *testing.T) {
	for _, tc := range []struct {
		Perms
		want string
	}{
		{0, "none"},
		{None, "none"},
		{Read, "read"},
		{Write, "write"},
		{Read | Write, "read,write"},
		{Write | Read, "read,write"},
		{Write | Read | None, "read,write"},
	} {
		if have, want := tc.String(), tc.want; have != want {
			t.Errorf(
				"Perms{%032b}:\nhave: %q\nwant: %q",
				tc.Perms,
				have, want,
			)
		}
	}
}

func BenchmarkPermsString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Read.String()
	}
}

func bitmap(ids ...uint32) *roaring.Bitmap {
	bm := roaring.NewBitmap()
	bm.AddMany(ids)
	return bm
}

func TestUserPermissions_AuthorizedRepos(t *testing.T) {
	tests := []struct {
		name     string
		repos    []*types.Repo
		p        *UserPermissions
		expPerms []RepoPerms
	}{
		{
			name:  "wrong permissions type",
			repos: []*types.Repo{},
			p: &UserPermissions{
				Perm: Read,
				Type: "",
				IDs:  bitmap(),
			},
			expPerms: []RepoPerms{},
		},
		{
			name:  "nil bitmap",
			repos: []*types.Repo{},
			p: &UserPermissions{
				Perm: Read,
				Type: PermRepos,
				IDs:  nil,
			},
			expPerms: []RepoPerms{},
		},
		{
			name:  "empty bitmap",
			repos: []*types.Repo{},
			p: &UserPermissions{
				Perm: Read,
				Type: PermRepos,
				IDs:  bitmap(),
			},
			expPerms: []RepoPerms{},
		},

		{
			name: "filter out repos have id=0",
			repos: []*types.Repo{
				{ID: 0},
				{ID: 1},
			},
			p: &UserPermissions{
				Perm: Read,
				Type: PermRepos,
				IDs:  bitmap(1),
			},
			expPerms: []RepoPerms{
				{Repo: &types.Repo{ID: 1}, Perms: Read},
			},
		},
		{
			name: "candidate list is a subset",
			repos: []*types.Repo{
				{ID: 1},
				{ID: 2},
			},
			p: &UserPermissions{
				Perm: Read,
				Type: PermRepos,
				IDs:  bitmap(1, 2, 3, 4),
			},
			expPerms: []RepoPerms{
				{Repo: &types.Repo{ID: 1}, Perms: Read},
				{Repo: &types.Repo{ID: 2}, Perms: Read},
			},
		},
		{
			name: "candidate list is a superset",
			repos: []*types.Repo{
				{ID: 1},
				{ID: 2},
				{ID: 3},
			},
			p: &UserPermissions{
				Perm: Read,
				Type: PermRepos,
				IDs:  bitmap(1, 2),
			},
			expPerms: []RepoPerms{
				{Repo: &types.Repo{ID: 1}, Perms: Read},
				{Repo: &types.Repo{ID: 2}, Perms: Read},
			},
		},
		{
			name: "candidate list has intersection",
			repos: []*types.Repo{
				{ID: 1},
				{ID: 2},
				{ID: 3},
			},
			p: &UserPermissions{
				Perm: Read,
				Type: PermRepos,
				IDs:  bitmap(1, 3, 5, 7),
			},
			expPerms: []RepoPerms{
				{Repo: &types.Repo{ID: 1}, Perms: Read},
				{Repo: &types.Repo{ID: 3}, Perms: Read},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			perms := test.p.AuthorizedRepos(test.repos)
			if diff := cmp.Diff(test.expPerms, perms); diff != "" {
				t.Fatalf("perms: %v", diff)
			}
		})
	}
}
