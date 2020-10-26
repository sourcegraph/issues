package main

func deduplicateIssues(issues []*Issue) (deduplicated []*Issue) {
	issuesMap := map[string]*Issue{}
	for _, v := range issues {
		issuesMap[v.ID] = v
	}

	for _, v := range issuesMap {
		deduplicated = append(deduplicated, v)
	}

	return deduplicated
}

func deduplicatePullRequests(pullRequests []*PullRequest) (deduplicated []*PullRequest) {
	prsMap := map[string]*PullRequest{}
	for _, v := range pullRequests {
		prsMap[v.ID] = v
	}

	for _, v := range prsMap {
		deduplicated = append(deduplicated, v)
	}

	return deduplicated
}
