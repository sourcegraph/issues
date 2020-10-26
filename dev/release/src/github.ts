import Octokit from '@octokit/rest'
import { readLine, formatDate, timezoneLink } from './util'
import { promisify } from 'util'
import * as semver from 'semver'
import { mkdtemp as original_mkdtemp } from 'fs'
import * as os from 'os'
import * as path from 'path'
import execa from 'execa'
import commandExists from 'command-exists'
const mkdtemp = promisify(original_mkdtemp)

function dateMarkdown(date: Date, name: string): string {
    return `[${formatDate(date)}](${timezoneLink(date, name)})`
}

export async function ensureTrackingIssue({
    majorVersion,
    minorVersion,
    assignees,
    releaseDateTime,
    oneWorkingDayBeforeRelease,
    fourWorkingDaysBeforeRelease,
    fiveWorkingDaysBeforeRelease,
    dryRun,
}: {
    majorVersion: string
    minorVersion: string
    assignees: string[]
    releaseDateTime: Date
    oneWorkingDayBeforeRelease: Date
    fourWorkingDaysBeforeRelease: Date
    fiveWorkingDaysBeforeRelease: Date
    dryRun: boolean
}): Promise<{ url: string; created: boolean }> {
    const octokit = await getAuthenticatedGitHubClient()
    const releaseIssueTemplate = await getContent(octokit, {
        owner: 'sourcegraph',
        repo: 'about',
        path: 'handbook/engineering/releases/release_issue_template.md',
    })
    const majorMinor = `${majorVersion}.${minorVersion}`
    const releaseIssueBody = releaseIssueTemplate
        .replace(/\$MAJOR/g, majorVersion)
        .replace(/\$MINOR/g, minorVersion)
        .replace(/\$RELEASE_DATE/g, dateMarkdown(releaseDateTime, `${majorMinor} release date`))
        .replace(
            /\$FIVE_WORKING_DAYS_BEFORE_RELEASE/g,
            dateMarkdown(fiveWorkingDaysBeforeRelease, `Five working days before ${majorMinor} release`)
        )
        .replace(
            /\$FOUR_WORKING_DAYS_BEFORE_RELEASE/g,
            dateMarkdown(fourWorkingDaysBeforeRelease, `Four working days before ${majorMinor} release`)
        )
        .replace(
            /\$ONE_WORKING_DAY_BEFORE_RELEASE/g,
            dateMarkdown(oneWorkingDayBeforeRelease, `One working day before ${majorMinor} release`)
        )

    const milestoneTitle = `${majorVersion}.${minorVersion}`
    const milestones = await octokit.issues.listMilestonesForRepo({
        owner: 'sourcegraph',
        repo: 'sourcegraph',
        per_page: 100,
        direction: 'desc',
    })
    const milestone = milestones.data.filter(milestone => milestone.title === milestoneTitle)
    if (milestone.length === 0) {
        console.log(
            `Milestone ${JSON.stringify(
                milestoneTitle
            )} is closed or not found—you'll need to manually create it and add this issue to it.`
        )
    }

    return ensureIssue(
        octokit,
        {
            title: trackingIssueTitle(majorVersion, minorVersion),
            owner: 'sourcegraph',
            repo: 'sourcegraph',
            assignees,
            body: releaseIssueBody,
            milestone: milestone.length > 0 ? milestone[0].number : undefined,
            labels: ['release-tracker'],
        },
        dryRun
    )
}

export async function ensurePatchReleaseIssue({
    version,
    assignees,
    dryRun,
}: {
    version: semver.SemVer
    assignees: string[]
    dryRun: boolean
}): Promise<{ url: string; created: boolean }> {
    const octokit = await getAuthenticatedGitHubClient()
    const issueTemplate = await getContent(octokit, {
        owner: 'sourcegraph',
        repo: 'about',
        path: 'handbook/engineering/releases/patch_release_issue_template.md',
    })
    const issueBody = issueTemplate
        .replace(/\$MAJOR/g, version.major.toString())
        .replace(/\$MINOR/g, version.minor.toString())
        .replace(/\$PATCH/g, version.patch.toString())
    return ensureIssue(
        octokit,
        {
            title: `${version.version} patch release`,
            owner: 'sourcegraph',
            repo: 'sourcegraph',
            assignees,
            body: issueBody,
        },
        dryRun
    )
}

async function getContent(
    octokit: Octokit,
    parameters: {
        owner: string
        repo: string
        path: string
    }
): Promise<string> {
    const response = await octokit.repos.getContents(parameters)
    if (Array.isArray(response.data)) {
        throw new TypeError(`${parameters.path} is a directory`)
    }
    return Buffer.from(response.data.content as string, 'base64').toString()
}

async function ensureIssue(
    octokit: Octokit,
    {
        title,
        owner,
        repo,
        assignees,
        body,
        milestone,
        labels,
    }: {
        title: string
        owner: string
        repo: string
        assignees: string[]
        body: string
        milestone?: number
        labels?: string[]
    },
    dryRun: boolean
): Promise<{ url: string; created: boolean }> {
    const issueData = {
        title,
        owner,
        repo,
        assignees,
        milestone,
        labels,
    }
    if (dryRun) {
        console.log('Dry run enabled, skipping issue creation')
        console.log(`Issue that would have been created:\n${JSON.stringify(issueData, null, 1)}`)
        console.log(`With body: ${body}`)
        return { url: '', created: false }
    }
    const url = await getIssueByTitle(octokit, title)
    if (url) {
        return { url, created: false }
    }
    const createdIssue = await octokit.issues.create({ body, ...issueData })
    return { url: createdIssue.data.html_url, created: true }
}

export async function listIssues(
    octokit: Octokit,
    query: string
): Promise<Octokit.SearchIssuesAndPullRequestsResponseItemsItem[]> {
    return (await octokit.search.issuesAndPullRequests({ per_page: 100, q: query })).data.items
}

export function trackingIssueTitle(major: string, minor: string): string {
    return `${major}.${minor} release tracking issue`
}

export async function getAuthenticatedGitHubClient(): Promise<Octokit> {
    const githubPAT = await readLine(
        'Enter a GitHub personal access token with "repo" scope (https://github.com/settings/tokens/new): ',
        '.secrets/github.txt'
    )
    const trimmedGithubPAT = githubPAT.trim()
    return new Octokit({ auth: trimmedGithubPAT })
}

export async function getIssueByTitle(octokit: Octokit, title: string): Promise<string | null> {
    const response = await octokit.search.issuesAndPullRequests({
        per_page: 100,
        q: `type:issue repo:sourcegraph/sourcegraph is:open ${JSON.stringify(title)}`,
    })

    const matchingIssues = response.data.items.filter(issue => issue.title === title)
    if (matchingIssues.length === 0) {
        return null
    }
    if (matchingIssues.length > 1) {
        throw new Error(`Multiple issues matched issue title ${JSON.stringify(title)}`)
    }
    return matchingIssues[0].html_url
}

export type EditFunc = (d: string) => void

export type Edit = string | EditFunc

export interface CreateBranchWithChangesOptions {
    owner: string
    repo: string
    base: string
    head: string
    commitMessage: string
    edits: Edit[]
    dryRun?: boolean
}

export interface ChangesetsOptions {
    requiredCommands: string[]
    changes: (Octokit.PullsCreateParams & CreateBranchWithChangesOptions)[]
    dryRun?: boolean
}

export interface CreatedChangeset {
    repository: string
    branch: string
    pullRequestURL: string
}

export async function createChangesets(options: ChangesetsOptions): Promise<CreatedChangeset[]> {
    for (const command of options.requiredCommands) {
        try {
            await commandExists(command)
        } catch {
            throw new Error(`Required command ${command} does not exist`)
        }
    }
    const octokit = await getAuthenticatedGitHubClient()

    // Generate changes
    const results: CreatedChangeset[] = []
    for (const change of options.changes) {
        await createBranchWithChanges(octokit, { ...change, dryRun: options.dryRun })
        let prURL = ''
        if (!options.dryRun) {
            prURL = await createPR(octokit, change)
        }
        results.push({
            repository: `${change.owner}/${change.repo}`,
            branch: change.base,
            pullRequestURL: prURL,
        })
    }

    // Log results
    for (const result of results) {
        console.log(`${result.repository} (${result.branch}): created pull request ${result.pullRequestURL}`)
    }

    return results
}

async function createBranchWithChanges(
    octokit: Octokit,
    { owner, repo, base: baseRevision, head: headBranch, commitMessage, edits, dryRun }: CreateBranchWithChangesOptions
): Promise<void> {
    const tmpdir = await mkdtemp(path.join(os.tmpdir(), `sg-release-${owner}-${repo}-`))
    console.log(`Created temp directory ${tmpdir}`)
    const depthFlag = '--depth 10'

    // Determine whether or not to create the base branch, or use the existing one
    let baseExists = true
    try {
        await octokit.repos.getBranch({ branch: baseRevision, owner, repo })
    } catch (error) {
        if (error.status === 404) {
            console.log(`Base ${baseRevision} does not exist`)
            baseExists = false
        } else {
            throw error
        }
    }
    const checkoutCommand =
        baseExists === true
            ? // check out the existing branch - fetch fails if we are already checked out, in which case just check out
              `git fetch ${depthFlag} origin ${baseRevision}:${baseRevision} || git checkout ${baseRevision}`
            : // create and publish base branch if it does not yet exist
              `git checkout -b ${baseRevision}`

    // Set up repository
    const setupScript = `set -ex

    git clone ${depthFlag} git@github.com:${owner}/${repo} || git clone ${depthFlag} https://github.com/${owner}/${repo};
    cd ./${repo};
    ${checkoutCommand};`
    await execa('bash', ['-c', setupScript], { stdio: 'inherit', cwd: tmpdir })
    const workdir = path.join(tmpdir, repo)

    // Apply edits
    for (const edit of edits) {
        switch (typeof edit) {
            case 'function':
                edit(workdir)
                break
            case 'string': {
                const editScript = `set -ex

                ${edit};`
                await execa('bash', ['-c', editScript], { stdio: 'inherit', cwd: workdir })
            }
        }
    }

    if (dryRun) {
        console.warn('Dry run enabled - printing diff instead of publishing')
        const showChangesScript = `set -ex

        git --no-pager diff;`
        await execa('bash', ['-c', showChangesScript], { stdio: 'inherit', cwd: workdir })
    } else {
        // Publish changes
        const publishScript = `set -ex

        git add :/;
        git commit -a -m ${JSON.stringify(commitMessage)};
        git push origin HEAD:${headBranch};`
        await execa('bash', ['-c', publishScript], { stdio: 'inherit', cwd: workdir })
    }
}

async function createPR(
    octokit: Octokit,
    options: {
        owner: string
        repo: string
        head: string
        base: string
        title: string
        body?: string
    }
): Promise<string> {
    const response = await octokit.pulls.create(options)
    return response.data.html_url
}
