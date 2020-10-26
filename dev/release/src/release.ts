import { ensureEvent, getClient, EventOptions } from './google-calendar'
import { postMessage } from './slack'
import {
    ensureTrackingIssue,
    getAuthenticatedGitHubClient,
    listIssues,
    getIssueByTitle,
    trackingIssueTitle,
    ensurePatchReleaseIssue,
    createChangesets,
} from './github'
import * as changelog from './changelog'
import { formatDate, timezoneLink } from './util'
import * as persistedConfig from './config.json'
import { addMinutes, isWeekend, eachDayOfInterval, addDays, subDays } from 'date-fns'
import * as semver from 'semver'
import execa from 'execa'
import { readFileSync, writeFileSync } from 'fs'
import * as path from 'path'

const sed = process.platform === 'linux' ? 'sed' : 'gsed'
interface Config {
    teamEmail: string

    captainSlackUsername: string
    captainGitHubUsername: string

    majorVersion: string
    minorVersion: string
    releaseDateTime: string
    oneWorkingDayBeforeRelease: string
    fourWorkingDaysBeforeRelease: string
    fiveWorkingDaysBeforeRelease: string

    slackAnnounceChannel: string

    dryRun: {
        changesets: boolean
        trackingIssues: boolean
    }
}

type StepID =
    | 'help'
    // release tracking
    | 'tracking:release-timeline'
    | 'tracking:release-issue'
    | 'tracking:patch-issue'
    // branch cut
    | 'changelog:cut'
    // release
    | 'release:status'
    | 'release:create-candidate'
    | 'release:publish'
    // testing
    | '_test:google-calendar'
    | '_test:slack'

interface Step {
    id: StepID
    run?: ((config: Config, ...args: string[]) => Promise<void>) | ((config: Config, ...args: string[]) => void)
    argNames?: string[]
}

const steps: Step[] = [
    {
        id: 'help',
        run: () => {
            console.error('Steps are:')
            console.error(
                steps
                    .filter(({ id }) => !id.startsWith('_'))
                    .map(
                        ({ id, argNames }) =>
                            '\t' +
                            id +
                            (argNames && argNames.length > 0
                                ? ' ' + argNames.map(argumentName => `<${argumentName}>`).join(' ')
                                : '')
                    )
                    .join('\n')
            )
        },
    },
    {
        id: '_test:google-calendar',
        run: async config => {
            const googleCalendar = await getClient()
            await ensureEvent(
                {
                    title: 'TEST EVENT',
                    startDateTime: new Date(config.releaseDateTime).toISOString(),
                    endDateTime: addMinutes(new Date(config.releaseDateTime), 1).toISOString(),
                },
                googleCalendar
            )
        },
    },
    {
        id: '_test:slack',
        run: async (_config, message) => {
            await postMessage(message, '_test-channel')
        },
    },
    {
        id: 'tracking:release-timeline',
        run: async config => {
            const googleCalendar = await getClient()
            const events: EventOptions[] = [
                {
                    title: 'Release captain: prepare for branch cut (5 working days until release)',
                    description: 'See the release tracking issue for TODOs',
                    startDateTime: new Date(config.fiveWorkingDaysBeforeRelease).toISOString(),
                    endDateTime: addMinutes(new Date(config.fiveWorkingDaysBeforeRelease), 1).toISOString(),
                },
                {
                    title: 'Release captain: branch cut (4 working days until release)',
                    description: 'See the release tracking issue for TODOs',
                    startDateTime: new Date(config.fourWorkingDaysBeforeRelease).toISOString(),
                    endDateTime: addMinutes(new Date(config.fourWorkingDaysBeforeRelease), 1).toISOString(),
                },
                ...eachDayOfInterval({
                    start: addDays(new Date(config.fourWorkingDaysBeforeRelease), 1),
                    end: subDays(new Date(config.oneWorkingDayBeforeRelease), 1),
                })
                    .filter(date => !isWeekend(date))
                    .map(date => ({
                        title: 'Release captain: cut new release candidate',
                        description: 'See release tracking issue for TODOs',
                        startDateTime: date.toISOString(),
                        endDateTime: addMinutes(date, 1).toISOString(),
                    })),
                {
                    title: 'Release captain: tag final release (1 working day before release)',
                    description: 'See the release tracking issue for TODOs',
                    startDateTime: new Date(config.oneWorkingDayBeforeRelease).toISOString(),
                    endDateTime: addMinutes(new Date(config.oneWorkingDayBeforeRelease), 1).toISOString(),
                },
                {
                    title: `Cut release branch ${config.majorVersion}.${config.minorVersion}`,
                    description: '(This is not an actual event to attend, just a calendar marker.)',
                    anyoneCanAddSelf: true,
                    attendees: [config.teamEmail],
                    startDateTime: new Date(config.fourWorkingDaysBeforeRelease).toISOString(),
                    endDateTime: addMinutes(new Date(config.fourWorkingDaysBeforeRelease), 1).toISOString(),
                },
                {
                    title: `Release Sourcegraph ${config.majorVersion}.${config.minorVersion}`,
                    description: '(This is not an actual event to attend, just a calendar marker.)',
                    anyoneCanAddSelf: true,
                    attendees: [config.teamEmail],
                    startDateTime: new Date(config.releaseDateTime).toISOString(),
                    endDateTime: addMinutes(new Date(config.releaseDateTime), 1).toISOString(),
                },
            ]

            for (const event of events) {
                console.log(`Create calendar event: ${event.title}: ${event.startDateTime || 'undefined'}`)
                await ensureEvent(event, googleCalendar)
            }
        },
    },
    {
        id: 'tracking:release-issue',
        run: async ({
            majorVersion,
            minorVersion,
            releaseDateTime,
            captainGitHubUsername,
            oneWorkingDayBeforeRelease,
            fourWorkingDaysBeforeRelease,
            fiveWorkingDaysBeforeRelease,

            captainSlackUsername,
            slackAnnounceChannel,
            dryRun,
        }: Config) => {
            // Create issue
            const { url, created } = await ensureTrackingIssue({
                majorVersion,
                minorVersion,
                assignees: [captainGitHubUsername],
                releaseDateTime: new Date(releaseDateTime),
                oneWorkingDayBeforeRelease: new Date(oneWorkingDayBeforeRelease),
                fourWorkingDaysBeforeRelease: new Date(fourWorkingDaysBeforeRelease),
                fiveWorkingDaysBeforeRelease: new Date(fiveWorkingDaysBeforeRelease),
                dryRun: dryRun.trackingIssues,
            })
            if (url) {
                console.log(created ? `Created tracking issue ${url}` : `Tracking issue already exists: ${url}`)
            }

            // Announce issue if issue does not already exist
            if (created) {
                // Slack markdown links
                const majorMinor = `${majorVersion}.${minorVersion}`
                const branchCutDate = new Date(fourWorkingDaysBeforeRelease)
                const branchCutDateString = `<${timezoneLink(branchCutDate, `${majorMinor} branch cut`)}|${formatDate(
                    branchCutDate
                )}>`
                const releaseDate = new Date(releaseDateTime)
                const releaseDateString = `<${timezoneLink(releaseDate, `${majorMinor} release`)}|${formatDate(
                    releaseDate
                )}>`
                await postMessage(
                    `*${majorVersion}.${minorVersion} Release*

:captain: Release captain: @${captainSlackUsername}
:pencil: Tracking issue: ${url}
:spiral_calendar_pad: Key dates:
* Branch cut: ${branchCutDateString}
* Release: ${releaseDateString}`,
                    slackAnnounceChannel
                )
                console.log(`Posted to Slack channel ${slackAnnounceChannel}`)
            }
        },
    },
    {
        id: 'tracking:patch-issue',
        run: async ({ captainGitHubUsername, slackAnnounceChannel, dryRun }, version) => {
            const parsedVersion = semver.parse(version, { loose: false })
            if (!parsedVersion) {
                throw new Error(`version ${version} is not valid semver`)
            }
            if (parsedVersion.prerelease.length > 0) {
                throw new Error(`version ${version} is pre-release`)
            }

            // Create issue
            const { url, created } = await ensurePatchReleaseIssue({
                version: parsedVersion,
                assignees: [captainGitHubUsername],
                dryRun: dryRun.trackingIssues,
            })
            if (url) {
                console.log(created ? `Created tracking issue ${url}` : `Tracking issue already exists: ${url}`)
            }

            // Announce issue if issue does not already exist
            if (created) {
                await postMessage(
                    `:captain: Patch release ${parsedVersion.version} will be published soon. If you have changes that should go into this patch release, please add your item to the checklist in the issue description: ${url}`,
                    slackAnnounceChannel
                )
                console.log(`Posted to Slack channel ${slackAnnounceChannel}`)
            }
        },
    },
    {
        id: 'changelog:cut',
        argNames: ['version', 'changelogFile'],
        run: async ({ dryRun }, version, changelogFile = 'CHANGELOG.md') => {
            const parsedVersion = semver.parse(version, { loose: false })
            if (!parsedVersion) {
                throw new Error(`version ${version} is not valid semver`)
            }

            await createChangesets({
                requiredCommands: [],
                changes: [
                    {
                        owner: 'sourcegraph',
                        repo: 'sourcegraph',
                        base: 'main',
                        head: `publish-${parsedVersion.version}`,
                        commitMessage: `release: sourcegraph@${parsedVersion.version}`,
                        edits: [
                            (directory: string) => {
                                console.log(`Updating '${changelogFile} for ${parsedVersion.format()}'`)
                                const changelogPath = path.join(directory, changelogFile)
                                let changelogContents = readFileSync(changelogPath).toString()

                                // Convert 'unreleased' to a release
                                const releaseHeader = `## ${parsedVersion.format()}`
                                const unreleasedHeader = '## Unreleased'
                                changelogContents = changelogContents.replace(unreleasedHeader, releaseHeader)

                                // Add a blank changelog template for the next release
                                changelogContents = changelogContents.replace(
                                    changelog.divider,
                                    changelog.releaseTemplate
                                )

                                // Update changelog
                                writeFileSync(changelogPath, changelogContents)
                            },
                        ], // Changes already done
                        title: `release: sourcegraph@${parsedVersion.version}`,
                    },
                ],
                dryRun: dryRun.changesets,
            })
        },
    },
    {
        id: 'release:status',
        run: async (config, version) => {
            const parsedVersion = semver.parse(version, { loose: false })
            if (!parsedVersion) {
                throw new Error(`version ${version} is not valid semver`)
            }

            const githubClient = await getAuthenticatedGitHubClient()

            const trackingIssueURL = await getIssueByTitle(
                githubClient,
                trackingIssueTitle(config.majorVersion, config.minorVersion)
            )
            if (!trackingIssueURL) {
                throw new Error(
                    `Tracking issue for version ${config.majorVersion}.${config.minorVersion} not found--has it been create yet?`
                )
            }

            const blockingQuery = 'is:open org:sourcegraph label:release-blocker'
            const blockingIssues = await listIssues(githubClient, blockingQuery)
            const blockingIssuesURL = `https://github.com/issues?q=${encodeURIComponent(blockingQuery)}`

            const openQuery = `is:open org:sourcegraph is:issue milestone:${config.majorVersion}.${config.minorVersion}`
            const openIssues = await listIssues(githubClient, openQuery)
            const openIssuesURL = `https://github.com/issues?q=${encodeURIComponent(openQuery)}`

            const issueCategories = [
                { name: 'release-blocking', issues: blockingIssues, issuesURL: blockingIssuesURL },
                { name: 'open', issues: openIssues, issuesURL: openIssuesURL },
            ]

            const message = `:captain: ${version} release status update:

- Tracking issue: ${trackingIssueURL}
${issueCategories
    .map(
        category =>
            '- ' +
            (category.issues.length === 1
                ? `There is 1 ${category.name} issue: ${category.issuesURL}`
                : `There are ${category.issues.length} ${category.name} issues: ${category.issuesURL}`)
    )
    .join('\n')}`
            await postMessage(message, config.slackAnnounceChannel)
        },
    },
    {
        id: 'release:create-candidate',
        argNames: ['version'],
        run: async (_config, version) => {
            const parsedVersion = semver.parse(version, { loose: false })
            if (!parsedVersion) {
                throw new Error(`version ${version} is not valid semver`)
            }
            const tag = JSON.stringify(`v${parsedVersion.version}`)
            const branch = JSON.stringify(`${parsedVersion.major}.${parsedVersion.minor}`)
            console.log(`Creating and pushing tag ${tag}`)
            await execa(
                'bash',
                [
                    '-c',
                    `git diff --quiet && git checkout ${branch} && git pull --rebase && git tag -a ${tag} -m ${tag} && git push origin ${tag}`,
                ],
                { stdio: 'inherit' }
            )
        },
    },
    {
        id: 'release:publish',
        run: async ({ slackAnnounceChannel, dryRun }, version) => {
            const parsedVersion = semver.parse(version, { loose: false })
            if (!parsedVersion) {
                throw new Error(`version ${version} is not valid semver`)
            }
            if (parsedVersion.prerelease.length > 0) {
                throw new Error(`version ${version} is pre-release`)
            }
            await createChangesets({
                requiredCommands: ['comby', sed, 'find', 'go'],
                changes: [
                    {
                        owner: 'sourcegraph',
                        repo: 'sourcegraph',
                        base: 'main',
                        head: `publish-${parsedVersion.version}`,
                        commitMessage: `release: sourcegraph@${parsedVersion.version}`,
                        title: `release: sourcegraph@${parsedVersion.version}`,
                        edits: [
                            `find . -type f -name '*.md' ! -name 'CHANGELOG.md' -exec ${sed} -i -E 's/sourcegraph\\/server:[0-9]+\\.[0-9]+\\.[0-9]+/sourcegraph\\/server:${parsedVersion.version}/g' {} +`,
                            `${sed} -i -E 's/version \`[0-9]+\\.[0-9]+\\.[0-9]+\`/version \`${parsedVersion.version}\`/g' doc/index.md`,
                            parsedVersion.patch === 0
                                ? `comby -in-place '{{$previousReleaseRevspec := ":[1]"}} {{$previousReleaseVersion := ":[2]"}} {{$currentReleaseRevspec := ":[3]"}} {{$currentReleaseVersion := ":[4]"}}' '{{$previousReleaseRevspec := ":[3]"}} {{$previousReleaseVersion := ":[4]"}} {{$currentReleaseRevspec := "v${parsedVersion.version}"}} {{$currentReleaseVersion := "${parsedVersion.major}.${parsedVersion.minor}"}}' doc/_resources/templates/document.html`
                                : `comby -in-place 'currentReleaseRevspec := ":[1]"' 'currentReleaseRevspec := "v${parsedVersion.version}"' doc/_resources/templates/document.html`,
                            `comby -in-place 'latestReleaseKubernetesBuild = newBuild(":[1]")' "latestReleaseKubernetesBuild = newBuild(\\"${parsedVersion.version}\\")" cmd/frontend/internal/app/updatecheck/handler.go`,
                            `comby -in-place 'latestReleaseDockerServerImageBuild = newBuild(":[1]")' "latestReleaseDockerServerImageBuild = newBuild(\\"${parsedVersion.version}\\")" cmd/frontend/internal/app/updatecheck/handler.go`,
                        ],
                    },
                    {
                        owner: 'sourcegraph',
                        repo: 'deploy-sourcegraph',
                        base: `${parsedVersion.major}.${parsedVersion.minor}`,
                        head: `publish-${parsedVersion.version}`,
                        commitMessage: `release: sourcegraph@${parsedVersion.version}`,
                        title: `release: sourcegraph@${parsedVersion.version}`,
                        edits: [
                            // installs version pinned by deploy-sourcegraph
                            'go install github.com/slimsag/update-docker-tags',
                            `.github/workflows/scripts/update-docker-tags.sh ${parsedVersion.version}`,
                        ],
                    },
                    {
                        owner: 'sourcegraph',
                        repo: 'deploy-sourcegraph-aws',
                        base: 'master',
                        head: `publish-${parsedVersion.version}`,
                        commitMessage: `release: sourcegraph@${parsedVersion.version}`,
                        title: `release: sourcegraph@${parsedVersion.version}`,
                        edits: [
                            `${sed} -i -E 's/export SOURCEGRAPH_VERSION=[0-9]+\\.[0-9]+\\.[0-9]+/export SOURCEGRAPH_VERSION=${parsedVersion.version}/g' resources/amazon-linux2.sh`,
                        ],
                    },
                    {
                        owner: 'sourcegraph',
                        repo: 'deploy-sourcegraph-digitalocean',
                        base: 'master',
                        head: `publish-${parsedVersion.version}`,
                        commitMessage: `release: sourcegraph@${parsedVersion.version}`,
                        title: `release: sourcegraph@${parsedVersion.version}`,
                        edits: [
                            `${sed} -i -E 's/export SOURCEGRAPH_VERSION=[0-9]+\\.[0-9]+\\.[0-9]+/export SOURCEGRAPH_VERSION=${parsedVersion.version}/g' resources/user-data.sh`,
                        ],
                    },
                ],
                dryRun: dryRun.changesets,
            })

            if (!dryRun.changesets) {
                await postMessage(
                    `${parsedVersion.version} has been released, update deploy-sourcegraph-docker as needed, cc @stephen`,
                    slackAnnounceChannel
                )
            }
        },
    },
]

async function run(config: Config, stepIDToRun: StepID, ...stepArguments: string[]): Promise<void> {
    await Promise.all(
        steps
            .filter(({ id }) => id === stepIDToRun)
            .map(async step => {
                if (step.run) {
                    await step.run(config, ...stepArguments)
                }
            })
    )
}

/**
 * Release captain automation
 */
async function main(): Promise<void> {
    const config = persistedConfig
    const args = process.argv.slice(2)
    if (args.length === 0) {
        console.error('This command expects at least 1 argument')
        await run(config, 'help')
        return
    }
    const step = args[0]
    if (!steps.map(({ id }) => id as string).includes(step)) {
        console.error('Unrecognized step', JSON.stringify(step))
        return
    }
    const stepArguments = args.slice(1)
    await run(config, step as StepID, ...stepArguments)
}

main().catch(error => console.error(error))
