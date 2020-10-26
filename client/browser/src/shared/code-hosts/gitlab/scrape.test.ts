import { getPageKindFromPathName, GitLabPageKind } from './scrape'

describe('getPageKindFromPathName()', () => {
    const TESTCASES: {
        title: string
        pathname: string
        owner: string
        projectName: string
        expected: GitLabPageKind
    }[] = [
        {
            title: 'blob page, dash in URL',
            pathname: '/gitlab-org/gitlab/-/blob/master/babel.config.js',
            owner: 'gitlab-org',
            projectName: 'gitlab',
            expected: GitLabPageKind.File,
        },
        {
            title: 'blob page, no dash in URL',
            pathname: '/gitlab-org/gitlab/blob/master/babel.config.js',
            owner: 'gitlab-org',
            projectName: 'gitlab',
            expected: GitLabPageKind.File,
        },
        {
            title: 'MR page, dash in URL',
            pathname: '/gitlab-org/gitlab/-/merge_requests/24675',
            owner: 'gitlab-org',
            projectName: 'gitlab',
            expected: GitLabPageKind.MergeRequest,
        },
        {
            title: 'MR page, no dash in URL',
            pathname: '/gitlab-org/gitlab/merge_requests/24675',
            owner: 'gitlab-org',
            projectName: 'gitlab',
            expected: GitLabPageKind.MergeRequest,
        },
        {
            title: 'Commit page, dash in URL',
            pathname: '/gitlab-org/gitlab/-/commit/2a4a5923cd71bdafce9699b4a16071255044b9f7',
            owner: 'gitlab-org',
            projectName: 'gitlab',
            expected: GitLabPageKind.Commit,
        },
        {
            title: 'Commit page, no dash in URL',
            pathname: '/gitlab-org/gitlab/commit/2a4a5923cd71bdafce9699b4a16071255044b9f7',
            owner: 'gitlab-org',
            projectName: 'gitlab',
            expected: GitLabPageKind.Commit,
        },
        {
            title: 'Project home',
            pathname: '/gitlab-org/gitlab',
            owner: 'gitlab-org',
            projectName: 'gitlab',
            expected: GitLabPageKind.Other,
        },
        {
            title: 'Project pipelines',
            pathname: '/gitlab-org/gitlab/pipelines',
            owner: 'gitlab-org',
            projectName: 'gitlab',
            expected: GitLabPageKind.Other,
        },
        {
            title: 'Merge requests list',
            pathname: '/gitlab-org/gitlab/-/merge_requests',
            owner: 'gitlab-org',
            projectName: 'gitlab',
            expected: GitLabPageKind.Other,
        },
    ]
    for (const { title, pathname, owner, projectName, expected } of TESTCASES) {
        test(title, () => {
            expect(getPageKindFromPathName(owner, projectName, pathname)).toBe(expected)
        })
    }
})
