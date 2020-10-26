/**
 * Defines configuration for e2e tests. This is as-yet incomplete as some config
 * depended on by other modules is not included here.
 */
export interface Config {
    browser: 'firefox' | 'chrome'
    sudoToken: string
    sudoUsername: string
    gitHubClientID: string
    gitHubClientSecret: string
    gitHubToken: string
    gitHubUserBobPassword: string
    gitHubUserBobToken: string
    gitHubUserAmyPassword: string
    gitLabToken: string
    gitLabClientID: string
    gitLabClientSecret: string
    gitLabUserAmyPassword: string
    oktaUserAmyPassword: string
    oktaMetadataUrl: string
    sourcegraphBaseUrl: string
    includeAdminOnboarding: boolean
    init: boolean
    testUserPassword: string
    noCleanup: boolean
    logStatusMessages: boolean
    logBrowserConsole: boolean
    slowMo: number
    devtools: boolean
    headless: boolean
    keepBrowser: boolean
    bitbucketCloudUserBobAppPassword: string
}

interface Field<T = string> {
    envVar: string
    description?: string
    defaultValue?: T
}

interface FieldParser<T = string> {
    parser: (rawValue: string) => T
}

type ConfigFields = {
    [K in keyof Config]: Field<Config[K]> & (Config[K] extends string ? Partial<FieldParser> : FieldParser<Config[K]>)
}

const parseBool = (string: string): boolean => {
    if (['1', 't', 'true'].some(trueString => trueString === string.toLowerCase())) {
        return true
    }
    if (['0', 'f', 'false'].some(falseString => falseString === string.toLowerCase())) {
        return false
    }
    throw new Error(`could not parse string ${JSON.stringify(string)} to boolean`)
}

const configFields: ConfigFields = {
    browser: {
        envVar: 'BROWSER',
        description: 'The browser to use.',
        defaultValue: 'chrome',
        parser: (value: string) => {
            if (!['firefox', 'chrome'].includes(value)) {
                throw new Error('BROWSER must be "chrome" or "firefox"')
            }
            return value
        },
    },
    sudoToken: {
        envVar: 'SOURCEGRAPH_SUDO_TOKEN',
        description:
            'An access token with "site-admin:sudo" permissions. This will be used to impersonate users in requests.',
    },
    sudoUsername: {
        envVar: 'SOURCEGRAPH_SUDO_USER',
        description: 'The site-admin-level username that will be impersonated with the sudo access token.',
    },
    gitHubClientID: {
        envVar: 'GITHUB_CLIENT_ID',
        description:
            'Client ID of the GitHub app to use to authenticate to Sourcegraph. Follow these instructions to obtain: https://developer.github.com/apps/building-oauth-apps/creating-an-oauth-app/',
        defaultValue: 'cf9491b706c4c3b1f956', // "Local dev sign-in via GitHub" OAuth app from github.com/sourcegraph
    },
    gitHubClientSecret: {
        envVar: 'GITHUB_CLIENT_SECRET',
        description:
            'Cilent secret of the GitHub app to use to authenticate to Sourcegraph. Follow these instructions to obtain: https://developer.github.com/apps/building-oauth-apps/creating-an-oauth-app/',
    },
    gitHubToken: {
        envVar: 'GITHUB_TOKEN',
        description:
            'A GitHub personal access token that will be used to authenticate a GitHub external service. It does not need to have any scopes.',
    },
    gitHubUserAmyPassword: {
        envVar: 'GITHUB_USER_AMY_PASSWORD',
        description: 'Password of the GitHub user sg-e2e-regression-test-amy, used to log in to Sourcegraph.',
    },
    gitHubUserBobPassword: {
        envVar: 'GITHUB_USER_BOB_PASSWORD',
        description: 'Password of the GitHub user sg-e2e-regression-test-bob, used to log in to Sourcegraph.',
    },
    gitHubUserBobToken: {
        envVar: 'GITHUB_USER_BOB_TOKEN',
        description:
            "GitHub personal access token with repo scope for user sg-e2e-regression-test-bob. Used to clone Bob's repositories to Sourcegraph.",
    },
    gitLabToken: {
        envVar: 'GITLAB_TOKEN',
        description:
            'A GitLab access token that will be used to authenticate a GitLab external service. It requires API scope.',
    },
    gitLabClientID: {
        envVar: 'GITLAB_CLIENT_ID',
        description:
            'Application ID of the GitLab OAuth app used to authenticate Sourcegraph. Follow these instructions to obtain: https://docs.gitlab.com/ee/integration/oauth_provider.html',
    },
    gitLabClientSecret: {
        envVar: 'GITLAB_CLIENT_SECRET',
        description:
            'Secret of the GitLab OAuth app used to authenticate Sourcegraph. Follow these instructions to obtain: https://docs.gitlab.com/ee/integration/oauth_provider.html',
    },
    gitLabUserAmyPassword: {
        envVar: 'GITLAB_USER_AMY_PASSWORD',
        description: 'Password of the GitLab user sg-e2e-regression-test-amy, used to log in to Sourcegraph.',
    },
    oktaMetadataUrl: {
        envVar: 'OKTA_METADATA_URL',
        description: 'URL of the Okta SAML IdP metadata.',
    },
    oktaUserAmyPassword: {
        envVar: 'OKTA_USER_AMY_PASSWORD',
        description: 'Password of the Okta user, beyang+sg-e2e-regression-test-amy@sourcegraph.com',
    },
    sourcegraphBaseUrl: {
        envVar: 'SOURCEGRAPH_BASE_URL',
        defaultValue: 'https://sourcegraph.test:3443',
        description:
            'The base URL of the Sourcegraph instance, e.g., https://sourcegraph.sgdev.org or https://sourcegraph.test:3443.',
    },
    includeAdminOnboarding: {
        envVar: 'INCLUDE_ADMIN_ONBOARDING',
        parser: parseBool,
        description:
            'If true, include admin onboarding tests, which assume none of the admin onboarding steps have yet completed on the instance. If those steps have already been completed, this test will fail.',
    },
    init: {
        envVar: 'E2E_INIT',
        parser: parseBool,
        defaultValue: false,
        description: 'If true, run the test for initializing a brand new instance of Sourcegraph',
    },
    testUserPassword: {
        envVar: 'TEST_USER_PASSWORD',
        description:
            'The password to use for any test users that are created. This password should be secure and unguessable when running against production Sourcegraph instances.',
    },
    noCleanup: {
        envVar: 'NO_CLEANUP',
        parser: parseBool,
        description:
            "If true, regression tests will not clean up users, external services, or other resources they create. Set this to true if running against a dev instance (as it'll make test runs faster). Set to false if running against production.",
    },
    keepBrowser: {
        envVar: 'KEEP_BROWSER',
        parser: parseBool,
        description: 'If true, browser window will remain open after tests run',
        defaultValue: false,
    },
    logBrowserConsole: {
        envVar: 'LOG_BROWSER_CONSOLE',
        parser: parseBool,
        description: "If false, don't log browser console to stdout.",
        defaultValue: true,
    },
    logStatusMessages: {
        envVar: 'LOG_STATUS_MESSAGES',
        parser: parseBool,
        description:
            'If true, logs status messages to console. This does not log the browser console (use LOG_BROWSER_CONSOLE for that).',
    },
    slowMo: {
        envVar: 'SLOWMO',
        parser: parseInt,
        description: 'Slow down puppeteer by the specified number of milliseconds',
        defaultValue: 0,
    },
    devtools: {
        envVar: 'DEVTOOLS',
        parser: parseBool,
        description: 'Run the tests with devtools open',
        defaultValue: false,
    },
    headless: {
        envVar: 'HEADLESS',
        parser: parseBool,
        description: 'Run Puppeteer in headless mode',
        defaultValue: false,
    },
    bitbucketCloudUserBobAppPassword: {
        envVar: 'BITBUCKET_CLOUD_USER_BOB_APP_PASSWORD',
        description:
            'A Bitbucket Cloud app password associated with the Bitbucket Cloud user sg-e2e-regression-test-bob, that will be used to sync Bitbucket Cloud repositories.',
    },
}

/**
 * Reads e2e config from environment variables. The caller should specify the config fields that it
 * depends on. This should NOT be called from helper packages. Instead, call it near the start of
 * "test main" function (i.e., Jest `test` blocks). Doing this ensures that all the necessary
 * environment variables necessary for a test are presented to the user in one go.
 */
export function getConfig<T extends keyof Config>(...required: T[]): Partial<Config> & Pick<Config, T> {
    // Read config
    const config: any = {}
    for (const fieldName of Object.keys(configFields) as (keyof Config)[]) {
        const field = configFields[fieldName]
        if (field.defaultValue !== undefined) {
            config[fieldName] = field.defaultValue
        }
        const environmentValue = process.env[field.envVar]
        if (environmentValue) {
            config[fieldName] = field.parser ? field.parser(environmentValue) : environmentValue
        }
    }

    // Check required fields for type safety
    const missingKeys = required.filter(key => config[key] === undefined)
    if (missingKeys.length > 0) {
        const fieldInfo = (key: T): string => {
            const field = configFields[key]
            if (!field) {
                return ''
            }
            const info = [field.envVar]
            if (field.defaultValue) {
                info.push(`default value: ${String(field.defaultValue)}`)
            }
            if (field.description) {
                info.push(`description: ${field.description}`)
            }
            return `${info.join(', ')}`
        }
        throw new Error(`FAIL: Required config was not provided. These environment variables were missing:

${missingKeys.map(key => `- ${fieldInfo(key)}`).join('\n')}

The recommended way to set them is to install direnv (https://direnv.net) and
create a .envrc file at the root of this repository.
        `)
    }

    return config
}
