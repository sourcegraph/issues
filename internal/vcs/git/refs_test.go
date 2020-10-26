package git

import (
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
)

func TestHumanReadableBranchName(t *testing.T) {
	for _, tc := range []struct {
		text string
		want string
	}{{
		// Respect word boundaries when cutting length
		text: "Change coördination mechanisms of fungible automation processes in place",
		want: "change-coordination-mechanisms-of-fungible-automation",
	}, {
		// Length smaller than maximum
		text: "Change coördination mechanisms",
		want: "change-coordination-mechanisms",
	}, {
		// Respecting word boundary would result in cutting too much,
		// so we don't.
		text: "Change alongwordmadeofmanylettersandnumbersandsymbolsandwhatnotisthisalreadymorethansixtyrunes",
		want: "change-alongwordmadeofmanylettersandnumbersandsymbolsandwhat",
	}} {
		if have := HumanReadableBranchName(tc.text); have != tc.want {
			t.Fatalf("HumanReadableBranchName(%q):\nhave %q\nwant %q", tc.text, have, tc.want)
		}
	}
}

func TestRepository_ListBranches(t *testing.T) {
	t.Parallel()

	gitCommands := []string{
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m foo --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"git checkout -b b0",
		"git checkout -b b1",
	}
	tests := map[string]struct {
		repo         gitserver.Repo
		wantBranches []*Branch
	}{
		"git cmd": {
			repo:         MakeGitRepository(t, gitCommands...),
			wantBranches: []*Branch{{Name: "b0", Head: "ea167fe3d76b1e5fd3ed8ca44cbd2fe3897684f8"}, {Name: "b1", Head: "ea167fe3d76b1e5fd3ed8ca44cbd2fe3897684f8"}, {Name: "master", Head: "ea167fe3d76b1e5fd3ed8ca44cbd2fe3897684f8"}},
		},
	}

	for label, test := range tests {
		branches, err := ListBranches(ctx, test.repo, BranchesOptions{})
		if err != nil {
			t.Errorf("%s: Branches: %s", label, err)
			continue
		}
		sort.Sort(Branches(branches))
		sort.Sort(Branches(test.wantBranches))

		if !reflect.DeepEqual(branches, test.wantBranches) {
			t.Errorf("%s: got branches == %v, want %v", label, AsJSON(branches), AsJSON(test.wantBranches))
		}
	}
}

func TestRepository_Branches_MergedInto(t *testing.T) {
	t.Parallel()

	gitCommands := []string{
		"git checkout -b b0",
		"echo 123 > some_other_file",
		"git add some_other_file",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -am foo --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -am foo --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",

		"git checkout HEAD^ -b b1",
		"git merge b0",

		"git checkout --orphan b2",
		"echo 234 > somefile",
		"git add somefile",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -am foo --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
	}

	gitBranches := map[string][]*Branch{
		"6520a4539a4cb664537c712216a53d80dd79bbdc": { // b1
			{Name: "b0", Head: "6520a4539a4cb664537c712216a53d80dd79bbdc"},
			{Name: "b1", Head: "6520a4539a4cb664537c712216a53d80dd79bbdc"},
		},
		"c3c691fc0fb1844a53b62b179e2fa9fdaf875718": { // b2
			{Name: "b2", Head: "c3c691fc0fb1844a53b62b179e2fa9fdaf875718"},
		},
	}

	for label, test := range map[string]struct {
		repo         gitserver.Repo
		wantBranches map[string][]*Branch
	}{
		"git cmd": {
			repo:         MakeGitRepository(t, gitCommands...),
			wantBranches: gitBranches,
		},
	} {
		for branch, mergedInto := range test.wantBranches {
			branches, err := ListBranches(ctx, test.repo, BranchesOptions{MergedInto: branch})
			if err != nil {
				t.Errorf("%s: Branches: %s", label, err)
				continue
			}
			if !cmp.Equal(mergedInto, branches) {
				t.Errorf("mismatch (-want +got):\n%s", cmp.Diff(mergedInto, branches))
			}
		}
	}
}

func TestRepository_Branches_ContainsCommit(t *testing.T) {
	t.Parallel()

	gitCommands := []string{
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m base --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m master --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"git checkout HEAD^ -b branch2",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m branch2 --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
	}

	// Pre-sorted branches
	gitWantBranches := map[string][]*Branch{
		"920c0e9d7b287b030ac9770fd7ba3ee9dc1760d9": {{Name: "branch2", Head: "920c0e9d7b287b030ac9770fd7ba3ee9dc1760d9"}},
		"1224d334dfe08f4693968ea618ad63ae86ec16ca": {{Name: "master", Head: "1224d334dfe08f4693968ea618ad63ae86ec16ca"}},
		"2816a72df28f699722156e545d038a5203b959de": {{Name: "branch2", Head: "920c0e9d7b287b030ac9770fd7ba3ee9dc1760d9"}, {Name: "master", Head: "1224d334dfe08f4693968ea618ad63ae86ec16ca"}},
	}

	tests := map[string]struct {
		repo                 gitserver.Repo
		commitToWantBranches map[string][]*Branch
	}{
		"git cmd": {
			repo:                 MakeGitRepository(t, gitCommands...),
			commitToWantBranches: gitWantBranches,
		},
	}

	for label, test := range tests {
		for commit, wantBranches := range test.commitToWantBranches {
			branches, err := ListBranches(ctx, test.repo, BranchesOptions{ContainsCommit: commit})
			if err != nil {
				t.Errorf("%s: Branches: %s", label, err)
				continue
			}

			sort.Sort(Branches(branches))
			if !reflect.DeepEqual(branches, wantBranches) {
				t.Errorf("%s: ContainsCommit %q: got branches == %v, want %v", label, commit, AsJSON(branches), AsJSON(wantBranches))
			}
		}
	}
}

func TestRepository_Branches_BehindAheadCounts(t *testing.T) {
	t.Parallel()

	gitCommands := []string{
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m foo0 --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"git branch old_work",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m foo1 --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m foo2 --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m foo3 --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m foo4 --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m foo5 --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"git checkout -b dev",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m foo6 --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m foo7 --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m foo8 --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"git checkout old_work",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m foo9 --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
	}
	gitBranches := []*Branch{
		{Counts: &BehindAhead{Behind: 5, Ahead: 1}, Name: "old_work", Head: "26692c614c59ddaef4b57926810aac7d5f0e94f0"},
		{Counts: &BehindAhead{Behind: 0, Ahead: 3}, Name: "dev", Head: "6724953367f0cd9a7755bac46ee57f4ab0c1aad8"},
		{Counts: &BehindAhead{Behind: 0, Ahead: 0}, Name: "master", Head: "8ea26e077a8fb9aa502c3fe2cfa3ce4e052d1a76"},
	}
	sort.Sort(Branches(gitBranches))

	tests := map[string]struct {
		repo         gitserver.Repo
		wantBranches []*Branch
	}{
		"git cmd": {
			repo:         MakeGitRepository(t, gitCommands...),
			wantBranches: gitBranches,
		},
	}

	for label, test := range tests {
		branches, err := ListBranches(ctx, test.repo, BranchesOptions{BehindAheadBranch: "master"})
		if err != nil {
			t.Errorf("%s: Branches: %s", label, err)
			continue
		}
		sort.Sort(Branches(branches))

		if !reflect.DeepEqual(branches, test.wantBranches) {
			t.Errorf("%s: got branches == %v, want %v", label, AsJSON(branches), AsJSON(test.wantBranches))
		}
	}
}

func TestRepository_Branches_IncludeCommit(t *testing.T) {
	t.Parallel()

	gitCommands := []string{
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m foo0 --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"git checkout -b b0",
		"GIT_COMMITTER_NAME=b GIT_COMMITTER_EMAIL=b@b.com GIT_COMMITTER_DATE=2006-01-02T15:04:06Z git commit --allow-empty -m foo1 --author='b <b@b.com>' --date 2006-01-02T15:04:06Z",
	}
	wantBranchesGit := []*Branch{
		{
			Name: "b0", Head: "c4a53701494d1d788b1ceeb8bf32e90224962473",
			Commit: &Commit{
				ID:        "c4a53701494d1d788b1ceeb8bf32e90224962473",
				Author:    Signature{Name: "b", Email: "b@b.com", Date: MustParseTime(time.RFC3339, "2006-01-02T15:04:06Z")},
				Committer: &Signature{Name: "b", Email: "b@b.com", Date: MustParseTime(time.RFC3339, "2006-01-02T15:04:06Z")},
				Message:   "foo1",
				Parents:   []api.CommitID{"a3c1537db9797215208eec56f8e7c9c37f8358ca"},
			},
		},
		{
			Name: "master", Head: "a3c1537db9797215208eec56f8e7c9c37f8358ca",
			Commit: &Commit{
				ID:        "a3c1537db9797215208eec56f8e7c9c37f8358ca",
				Author:    Signature{Name: "a", Email: "a@a.com", Date: MustParseTime(time.RFC3339, "2006-01-02T15:04:05Z")},
				Committer: &Signature{Name: "a", Email: "a@a.com", Date: MustParseTime(time.RFC3339, "2006-01-02T15:04:05Z")},
				Message:   "foo0",
				Parents:   nil,
			},
		},
	}

	tests := map[string]struct {
		repo         gitserver.Repo
		wantBranches []*Branch
	}{
		"git cmd": {
			repo:         MakeGitRepository(t, gitCommands...),
			wantBranches: wantBranchesGit,
		},
	}

	for label, test := range tests {
		branches, err := ListBranches(ctx, test.repo, BranchesOptions{IncludeCommit: true})
		if err != nil {
			t.Errorf("%s: Branches: %s", label, err)
			continue
		}
		sort.Sort(Branches(branches))

		if !reflect.DeepEqual(branches, test.wantBranches) {
			t.Errorf("%s: got branches == %v, want %v", label, AsJSON(branches), AsJSON(test.wantBranches))
		}
	}
}

func TestRepository_ListTags(t *testing.T) {
	t.Parallel()

	dateEnv := "GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z"
	gitCommands := []string{
		dateEnv + " git commit --allow-empty -m foo --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"git tag t0",
		"git tag t1",
		dateEnv + " git tag --annotate -m foo t2",
	}
	tests := map[string]struct {
		repo     gitserver.Repo
		wantTags []*Tag
	}{
		"git cmd": {
			repo: MakeGitRepository(t, gitCommands...),
			wantTags: []*Tag{
				{Name: "t0", CommitID: "ea167fe3d76b1e5fd3ed8ca44cbd2fe3897684f8", CreatorDate: MustParseTime(time.RFC3339, "2006-01-02T15:04:05Z")},
				{Name: "t1", CommitID: "ea167fe3d76b1e5fd3ed8ca44cbd2fe3897684f8", CreatorDate: MustParseTime(time.RFC3339, "2006-01-02T15:04:05Z")},
				{Name: "t2", CommitID: "ea167fe3d76b1e5fd3ed8ca44cbd2fe3897684f8", CreatorDate: MustParseTime(time.RFC3339, "2006-01-02T15:04:05Z")},
			},
		},
	}

	for label, test := range tests {
		tags, err := ListTags(ctx, test.repo)
		if err != nil {
			t.Errorf("%s: ListTags: %s", label, err)
			continue
		}

		sort.Sort(Tags(tags))
		sort.Sort(Tags(test.wantTags))

		if !reflect.DeepEqual(tags, test.wantTags) {
			t.Errorf("%s: got tags == %v, want %v", label, tags, test.wantTags)
		}
	}
}

// See https://github.com/sourcegraph/sourcegraph/issues/5453
func TestRepository_parseTags_WithoutCreatorDate(t *testing.T) {
	have, err := parseTags([]byte(
		"9ee1c939d1cb936b1f98e8d81aeffab57bae46ab\x00v2.6.12\x001119037709\n" +
			"c39ae07f393806ccf406ef966e9a15afc43cc36a\x00v2.6.11-tree\x00\n" +
			"c39ae07f393806ccf406ef966e9a15afc43cc36a\x00v2.6.11\x00\n",
	))

	if err != nil {
		t.Fatalf("parseTags: have err %v, want nil", err)
	}

	want := []*Tag{
		{
			Name:        "v2.6.12",
			CommitID:    "9ee1c939d1cb936b1f98e8d81aeffab57bae46ab",
			CreatorDate: time.Unix(1119037709, 0).UTC(),
		},
		{
			Name:     "v2.6.11-tree",
			CommitID: "c39ae07f393806ccf406ef966e9a15afc43cc36a",
		},
		{
			Name:     "v2.6.11",
			CommitID: "c39ae07f393806ccf406ef966e9a15afc43cc36a",
		},
	}

	if diff := cmp.Diff(have, want); diff != "" {
		t.Fatal(diff)
	}
}

func TestValidateBranchName(t *testing.T) {
	for _, tc := range []struct {
		name   string
		branch string
		valid  bool
	}{
		{name: "Valid branch", branch: "valid-branch", valid: true},
		{name: "Valid branch with slash", branch: "rgs/valid-branch", valid: true},
		{name: "Valid branch with @", branch: "valid@branch", valid: true},
		{name: "Path component with .", branch: "valid-/.branch", valid: false},
		{name: "Double dot", branch: "valid..branch", valid: false},
		{name: "End with .lock", branch: "valid-branch.lock", valid: false},
		{name: "No space", branch: "valid branch", valid: false},
		{name: "No tilde", branch: "valid~branch", valid: false},
		{name: "No carat", branch: "valid^branch", valid: false},
		{name: "No colon", branch: "valid:branch", valid: false},
		{name: "No question mark", branch: "valid?branch", valid: false},
		{name: "No asterisk", branch: "valid*branch", valid: false},
		{name: "No open bracket", branch: "valid[branch", valid: false},
		{name: "No trailing slash", branch: "valid-branch/", valid: false},
		{name: "No beginning slash", branch: "/valid-branch", valid: false},
		{name: "No double slash", branch: "valid//branch", valid: false},
		{name: "No trailing dot", branch: "valid-branch.", valid: false},
		{name: "Cannot contain @{", branch: "valid@{branch", valid: false},
		{name: "Cannot be @", branch: "@", valid: false},
		{name: "Cannot contain backslash", branch: "valid\\branch", valid: false},
		{name: "head not allowed", branch: "head", valid: false},
		{name: "Head not allowed", branch: "Head", valid: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			valid := ValidateBranchName(tc.branch)
			if tc.valid != valid {
				t.Fatalf("Expected %t, got %t", tc.valid, valid)
			}
		})
	}
}
