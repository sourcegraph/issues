import { readFileSync, unlinkSync } from 'fs'

import { parse as parseJSONC } from '@sqs/jsonc-parser'
import * as semver from 'semver'

import { cacheFolder, readLine, getWeekNumber } from './util'

/**
 * Release configuration file format
 */
export interface Config {
    teamEmail: string

    captainSlackUsername: string
    captainGitHubUsername: string

    previousRelease: string
    upcomingRelease: string

    oneWorkingDayBeforeRelease: string
    releaseDate: string
    oneWorkingDayAfterRelease: string

    slackAnnounceChannel: string

    dryRun: {
        tags?: boolean
        changesets?: boolean
        trackingIssues?: boolean
        calendar?: boolean
    }
}

/**
 * Default path of JSONC containing release configuration.
 */
const configPath = 'release-config.jsonc'

/**
 * Loads configuration from predefined path. It does not do any special validation.
 */
export function loadConfig(): Config {
    return parseJSONC(readFileSync(configPath).toString()) as Config
}

/**
 * Convenience function for getting relevant configured releases as semver.SemVer
 *
 * It prompts for a confirmation of the `upcomingRelease` that is cached for a week.
 */
export async function releaseVersions(
    config: Config
): Promise<{
    previous: semver.SemVer
    upcoming: semver.SemVer
}> {
    const parseOptions: semver.Options = { loose: false }
    const parsedPrevious = semver.parse(config.previousRelease, parseOptions)
    if (!parsedPrevious) {
        throw new Error(`config.previousRelease '${config.previousRelease}' is not valid semver`)
    }
    const parsedUpcoming = semver.parse(config.upcomingRelease, parseOptions)
    if (!parsedUpcoming) {
        throw new Error(`config.upcomingRelease '${config.upcomingRelease}' is not valid semver`)
    }

    // Verify the configured upcoming release. The response is cached and expires in a
    // week, after which the captain is required to confirm again.
    const now = new Date()
    const cachedVersionResponse = `${cacheFolder}/current_release_${now.getUTCFullYear()}_${getWeekNumber(now)}.txt`
    const confirmVersion = await readLine(
        `Please confirm the upcoming release version configured in '${configPath}' (currently '${config.upcomingRelease}'): `,
        cachedVersionResponse
    )
    const parsedConfirmed = semver.parse(confirmVersion, parseOptions)
    let error = ''
    if (!parsedConfirmed) {
        error = `Provided version '${confirmVersion}' is not valid semver`
    } else if (semver.neq(parsedConfirmed, parsedUpcoming)) {
        error = `Provided version '${confirmVersion}' and config.upcomingRelease '${config.upcomingRelease}' do not match - please update the release configuration at '${configPath}' and try again`
    }

    // If error, abort and remove the cached response (since it is invalid anyway)
    if (error !== '') {
        unlinkSync(cachedVersionResponse)
        throw new Error(error)
    }

    const versions = {
        previous: parsedPrevious,
        upcoming: parsedUpcoming,
    }
    console.log(`Using versions: { upcoming: ${versions.upcoming.format()}, previous: ${versions.previous.format()} }`)
    return versions
}
