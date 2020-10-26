import { describe, test, before, beforeEach, after, afterEach } from 'mocha'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import { afterEachRecordCoverage } from '../../../shared/src/testing/coverage'
import { retry } from '../../../shared/src/testing/utils'
import { createDriverForTest, Driver, percySnapshot } from '../../../shared/src/testing/driver'
import got from 'got'
import { gql } from '../../../shared/src/graphql/graphql'
import { random, sortBy } from 'lodash'
import MockDate from 'mockdate'
import { ExternalServiceKind } from '../../../shared/src/graphql/schema'
import { getConfig } from '../../../shared/src/testing/config'
import assert from 'assert'
import expect from 'expect'
import { Settings } from '../schema/settings.schema'

const { gitHubToken, sourcegraphBaseUrl } = getConfig('gitHubToken', 'sourcegraphBaseUrl')

describe('e2e test suite', () => {
    let driver: Driver

    before(async function () {
        // Cloning the repositories takes ~1 minute, so give initialization 2
        // minutes instead of 1 (which would be inherited from
        // `jest.setTimeout(1 * 60 * 1000)` above).
        this.timeout(2 * 60 * 1000)

        // Reset date mocking
        MockDate.reset()

        const config = getConfig('headless', 'slowMo', 'testUserPassword')

        // Start browser
        driver = await createDriverForTest({
            sourcegraphBaseUrl,
            logBrowserConsole: true,
            ...config,
        })
        const clonedRepoSlugs = [
            'sourcegraph/java-langserver',
            'gorilla/mux',
            'gorilla/securecookie',
            'sourcegraph/jsonrpc2',
            'sourcegraph/go-diff',
            'sourcegraph/appdash',
            'sourcegraph/sourcegraph-typescript',
            'sourcegraph-testing/automation-e2e-test',
            'sourcegraph/e2e-test-private-repository',
        ]
        const alwaysCloningRepoSlugs = ['sourcegraphtest/AlwaysCloningTest']
        await driver.ensureLoggedIn({ username: 'test', password: config.testUserPassword, email: 'test@test.com' })
        await driver.resetUserSettings()
        await driver.ensureHasExternalService({
            kind: ExternalServiceKind.GITHUB,
            displayName: 'test-test-github',
            config: JSON.stringify({
                url: 'https://github.com',
                token: gitHubToken,
                repos: clonedRepoSlugs.concat(alwaysCloningRepoSlugs),
            }),
            ensureRepos: clonedRepoSlugs.map(slug => `github.com/${slug}`),
            alwaysCloning: alwaysCloningRepoSlugs.map(slug => `github.com/${slug}`),
        })
    })

    after('Close browser', () => driver?.close())

    afterEachSaveScreenshotIfFailed(() => driver.page)
    afterEachRecordCoverage(() => driver)

    beforeEach(async () => {
        if (driver) {
            // Clear local storage to reset sidebar selection (files or tabs) for each test
            await driver.page.evaluate(() => {
                localStorage.setItem('repo-revision-sidebar-last-tab', 'files')
            })

            await driver.resetUserSettings()
        }
    })

    describe('Core functionality', () => {
        test('Check settings are saved and applied', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/users/test/settings')
            await driver.page.waitForSelector('.test-settings-file .monaco-editor')

            const message = 'A wild notice appears!'
            await driver.replaceText({
                selector: '.test-settings-file .monaco-editor',
                newText: JSON.stringify({
                    notices: [
                        {
                            dismissable: false,
                            location: 'top',
                            message,
                        },
                    ],
                }),
                selectMethod: 'keyboard',
            })
            await driver.page.click('.test-settings-file .test-save-toolbar-save')
            await driver.page.waitForSelector('.test-global-alert .notices .global-alerts__alert', { visible: true })
            await driver.page.evaluate((message: string) => {
                const element = document.querySelector<HTMLElement>('.test-global-alert .notices .global-alerts__alert')
                if (!element) {
                    throw new Error('No .test-global-alert .notices .global-alerts__alert element found')
                }
                if (!element.textContent?.includes(message)) {
                    throw new Error(`Expected "${message}" message, but didn't find it`)
                }
            }, message)
        })

        test('Check access tokens work (create, use and delete)', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/users/test/settings/tokens/new')
            await driver.page.waitForSelector('.test-create-access-token-description')

            const name = `E2E Test ${new Date().toISOString()} ${random(1, 1e7)}`

            await driver.replaceText({
                selector: '.test-create-access-token-description',
                newText: name,
                selectMethod: 'keyboard',
            })

            await driver.page.click('.test-create-access-token-submit')
            const token = (await (
                await driver.page.waitForFunction(
                    () => document.querySelector<HTMLInputElement>('.test-access-token input[type=text]')?.value
                )
            ).jsonValue()) as string | null
            assert(token)

            const response = await got.post('.api/graphql', {
                prefixUrl: sourcegraphBaseUrl,
                headers: {
                    Authorization: 'token ' + token,
                },
                body: JSON.stringify({
                    query: gql`
                        query {
                            currentUser {
                                username
                            }
                        }
                    `,
                    variables: {},
                }),
            })

            const username = JSON.parse(response.body).data.currentUser.username
            expect(username).toBe('test')

            await Promise.all([
                driver.acceptNextDialog(),
                (
                    await driver.page.waitForSelector(
                        `[data-test-access-token-description="${name}"] .test-access-token-delete`,
                        { visible: true }
                    )
                ).click(),
            ])

            await driver.page.waitFor(
                (name: string) => !document.querySelector(`[data-test-access-token-description="${name}"]`),
                {},
                name
            )
        })

        test('Check allowed usernames', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/users/test/settings/profile')
            await driver.page.waitForSelector('.test-UserProfileFormFields-username')

            const name = 'alice.bob-chris-'

            await driver.replaceText({
                selector: '.test-UserProfileFormFields-username',
                newText: name,
                selectMethod: 'selectall',
            })

            await driver.page.click('#test-EditUserProfileForm__save')
            await driver.page.waitForSelector('.test-EditUserProfileForm__success', { visible: true })

            await driver.page.goto(sourcegraphBaseUrl + `/users/${name}/settings/profile`)
            await driver.replaceText({
                selector: '.test-UserProfileFormFields-username',
                newText: 'test',
                selectMethod: 'selectall',
            })

            await driver.page.click('#test-EditUserProfileForm__save')
            await driver.page.waitForSelector('.test-EditUserProfileForm__success', { visible: true })
        })
    })

    describe('External services', () => {
        test('External service add, edit, delete', async () => {
            const displayName = 'test-github-test-2'
            await driver.ensureHasExternalService({
                kind: ExternalServiceKind.GITHUB,
                displayName,
                config:
                    '{"url": "https://github.myenterprise.com", "token": "initial-token", "repositoryQuery": ["none"]}',
            })
            await driver.page.goto(sourcegraphBaseUrl + '/site-admin/external-services')
            await (
                await driver.page.waitForSelector(
                    `[data-test-external-service-name="${displayName}"] .test-edit-external-service-button`
                )
            ).click()

            // Type in a new external service configuration.
            await driver.replaceText({
                selector: '.test-external-service-editor .monaco-editor',
                newText:
                    '{"url": "https://github.myenterprise.com", "token": "second-token", "repositoryQuery": ["none"]}',
                selectMethod: 'selectall',
                enterTextMethod: 'paste',
            })
            await driver.page.click('.test-update-external-service-button')
            // Must wait for the operation to complete, or else a "Discard changes?" dialog will pop up
            await driver.page.waitForSelector('.test-update-external-service-button:not([disabled])', { visible: true })

            await (
                await driver.page.waitForSelector('.list-group-item[href="/site-admin/external-services"]', {
                    visible: true,
                })
            ).click()

            await Promise.all([
                driver.acceptNextDialog(),
                (
                    await driver.page.waitForSelector(
                        '[data-test-external-service-name="test-github-test-2"] .test-delete-external-service-button',
                        { visible: true }
                    )
                ).click(),
            ])

            await driver.page.waitFor(
                () => !document.querySelector('[data-test-external-service-name="test-github-test-2"]')
            )
        })

        test('External service repositoryPathPattern', async () => {
            const repo = 'sourcegraph/go-blame' // Tiny repo, fast to clone
            const repositoryPathPattern = 'foobar/{host}/{nameWithOwner}'
            const slug = `github.com/${repo}`
            const pathPatternSlug = `foobar/github.com/${repo}`

            const config = {
                kind: ExternalServiceKind.GITHUB,
                displayName: 'test-test-github-repoPathPattern',
                config: JSON.stringify({
                    url: 'https://github.com',
                    token: gitHubToken,
                    repos: [repo],
                    repositoryPathPattern,
                }),
                // Make sure repository is named according to path pattern
                ensureRepos: [pathPatternSlug],
            }
            await driver.ensureHasExternalService(config)

            // Make sure repository slug without path pattern redirects to path pattern
            await driver.page.goto(sourcegraphBaseUrl + '/' + slug)
            await driver.assertWindowLocationPrefix('/' + pathPatternSlug)
        })

        const awsAccessKeyID = process.env.AWS_ACCESS_KEY_ID
        const awsSecretAccessKey = process.env.AWS_SECRET_ACCESS_KEY
        const awsCodeCommitUsername = process.env.AWS_CODE_COMMIT_GIT_USERNAME
        const awsCodeCommitPassword = process.env.AWS_CODE_COMMIT_GIT_PASSWORD

        const testIfAwsCredentialsSet =
            awsSecretAccessKey && awsAccessKeyID && awsCodeCommitUsername && awsCodeCommitPassword
                ? test
                : test.skip.bind(test)

        testIfAwsCredentialsSet('AWS CodeCommit', async () => {
            await driver.ensureHasExternalService({
                kind: ExternalServiceKind.AWSCODECOMMIT,
                displayName: 'test-aws-code-commit',
                config: JSON.stringify({
                    region: 'us-west-1',
                    accessKeyID: awsAccessKeyID,
                    secretAccessKey: awsSecretAccessKey,
                    repositoryPathPattern: 'aws/{name}',
                    gitCredentials: {
                        username: awsCodeCommitUsername,
                        password: awsCodeCommitPassword,
                    },
                }),
                ensureRepos: ['aws/test'],
            })
            await driver.page.goto(sourcegraphBaseUrl + '/aws/test/-/blob/README')
            const blob = (await (
                await driver.page.waitFor(() => document.querySelector<HTMLElement>('.test-repo-blob')?.textContent)
            ).jsonValue()) as string | null

            expect(blob).toBe('README\n\nchange')
        })

        const bbsURL = process.env.BITBUCKET_SERVER_URL
        const bbsToken = process.env.BITBUCKET_SERVER_TOKEN
        const bbsUsername = process.env.BITBUCKET_SERVER_USERNAME

        const testIfBBSCredentialsSet = bbsURL && bbsToken && bbsUsername ? test : test.skip.bind(test)

        testIfBBSCredentialsSet('Bitbucket Server', async () => {
            await driver.ensureHasExternalService({
                kind: ExternalServiceKind.BITBUCKETSERVER,
                displayName: 'test-bitbucket-server',
                config: JSON.stringify({
                    url: bbsURL,
                    token: bbsToken,
                    username: bbsUsername,
                    repos: ['SOURCEGRAPH/jsonrpc2'],
                    repositoryPathPattern: 'bbs/{projectKey}/{repositorySlug}',
                }),
                ensureRepos: ['bbs/SOURCEGRAPH/jsonrpc2'],
            })
            await driver.page.goto(sourcegraphBaseUrl + '/bbs/SOURCEGRAPH/jsonrpc2/-/blob/.travis.yml')
            const blob = (await (
                await driver.page.waitFor(() => document.querySelector<HTMLElement>('.test-repo-blob')?.textContent)
            ).jsonValue()) as string | null

            expect(blob).toBe('language: go\ngo: \n - 1.x\n\nscript:\n - go test -race -v ./...')
        })
    })

    describe('Visual tests', () => {
        test('Repositories list', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/site-admin/repositories?query=gorilla%2Fmux')
            await driver.page.waitForSelector('a[href="/github.com/gorilla/mux"]', { visible: true })
            await percySnapshot(driver.page, 'Repositories list')
        })

        test('Search results repo', async () => {
            await driver.page.goto(
                sourcegraphBaseUrl + '/search?q=repo:%5Egithub.com/gorilla/mux%24&patternType=regexp'
            )
            await driver.page.waitForSelector('a[href="/github.com/gorilla/mux"]', { visible: true })
            // Flaky https://github.com/sourcegraph/sourcegraph/issues/2704
            // await percySnapshot(page, 'Search results repo')
        })

        test('Search results file', async () => {
            await driver.page.goto(
                sourcegraphBaseUrl + '/search?q=repo:%5Egithub.com/gorilla/mux%24+file:%5Emux.go%24&patternType=regexp'
            )
            await driver.page.waitForSelector('a[href="/github.com/gorilla/mux"]', { visible: true })
            // Flaky https://github.com/sourcegraph/sourcegraph/issues/2704
            // await percySnapshot(page, 'Search results file')
        })

        test('Search visibility:private|public', async () => {
            const privateRepos = ['github.com/sourcegraph/e2e-test-private-repository']

            await driver.page.goto(sourcegraphBaseUrl + '/search?q=type:repo+visibility:private')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length >= 1)

            const privateResults = await driver.page.evaluate(() =>
                [...document.querySelectorAll('.test-search-result-label')].map(label =>
                    (label.textContent || '').trim()
                )
            )
            expect(privateResults).toEqual(expect.arrayContaining(privateRepos))

            await driver.page.goto(sourcegraphBaseUrl + '/search?q=type:repo+visibility:public')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 1)

            const publicResults = await driver.page.evaluate(() =>
                [...document.querySelectorAll('.etest-search-result-label')].map(label =>
                    (label.textContent || '').trim()
                )
            )
            expect(publicResults).not.toEqual(expect.arrayContaining(privateRepos))

            await driver.page.goto(sourcegraphBaseUrl + '/search?q=type:repo+visibility:any')
            await driver.page.waitForFunction(() => document.querySelectorAll('.test-search-result').length > 1)

            const anyResults = await driver.page.evaluate(() =>
                [...document.querySelectorAll('.test-search-result-label')].map(label =>
                    (label.textContent || '').trim()
                )
            )
            expect(anyResults).toEqual(expect.arrayContaining(privateRepos))
        })

        test('Search results code', async () => {
            await driver.page.goto(
                sourcegraphBaseUrl +
                    '/search?q=repo:^github.com/gorilla/mux$&patternType=regexp file:mux.go "func NewRouter"'
            )
            await driver.page.waitForSelector('a[href="/github.com/gorilla/mux"]', { visible: true })
            // Flaky https://github.com/sourcegraph/sourcegraph/issues/2704
            // await percySnapshot(page, 'Search results code')
        })

        test('Site admin overview', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/site-admin')
            await driver.page.waitForSelector('.test-site-admin-overview-menu', { visible: true })
            await driver.page.waitForSelector('.test-product-certificate', { visible: true })
            await percySnapshot(driver.page, 'Site admin overview')
        })
    })

    describe('Theme switcher', () => {
        test('changes the theme', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/github.com/gorilla/mux/-/blob/mux.go')
            await driver.page.waitForSelector('.theme.theme-dark, .theme.theme-light', { visible: true })

            const getActiveThemeClasses = (): Promise<string[]> =>
                driver.page.evaluate(() =>
                    [...document.querySelector('.theme')!.classList].filter(className => className.startsWith('theme-'))
                )

            expect(await getActiveThemeClasses()).toHaveLength(1)
            await driver.page.waitForSelector('.test-user-nav-item-toggle')
            await driver.page.click('.test-user-nav-item-toggle')

            // Switch to dark
            await driver.page.select('.test-theme-toggle', 'dark')
            expect(await getActiveThemeClasses()).toEqual(['theme-dark'])

            // Switch to light
            await driver.page.select('.test-theme-toggle', 'light')
            expect(await getActiveThemeClasses()).toEqual(['theme-light'])
        })
    })

    describe('Repository component', () => {
        const blobTableSelector = '.test-blob > table'
        /**
         * @param line 1-indexed line number
         * @param spanOffset 1-indexed index of the span that's to be clicked
         */
        const clickToken = async (line: number, spanOffset: number): Promise<void> => {
            const selector = `${blobTableSelector} tr:nth-child(${line}) > td.code > div:nth-child(1) > span:nth-child(${spanOffset})`
            await driver.page.waitForSelector(selector, { visible: true })
            await driver.page.click(selector)
        }

        const getHoverContents = async (): Promise<string[]> => {
            // Search for any child of test-tooltip-content: as test-tooltip-content has display: contents,
            // it will never be detected as visible by waitForSelector(), but its children will.
            const selector = '.test-tooltip-content *'
            await driver.page.waitForSelector(selector, { visible: true })
            return driver.page.evaluate(() =>
                // You can't reference hoverContentSelector in puppeteer's driver.page.evaluate
                [...document.querySelectorAll('.test-tooltip-content')].map(content => content.textContent || '')
            )
        }
        const assertHoverContentContains = async (value: string): Promise<void> => {
            expect(await getHoverContents()).toEqual(expect.arrayContaining([expect.stringContaining(value)]))
        }

        const clickHoverJ2D = async (): Promise<void> => {
            const selector = '.test-tooltip-go-to-definition'
            await driver.page.waitForSelector(selector, { visible: true })
            await driver.page.click(selector)
        }
        const clickHoverFindReferences = async (): Promise<void> => {
            const selector = '.test-tooltip-find-references'
            await driver.page.waitForSelector(selector, { visible: true })
            await driver.page.click(selector)
        }

        describe('file tree', () => {
            test('does navigation on file click', async () => {
                await driver.page.goto(
                    sourcegraphBaseUrl + '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d'
                )
                await (
                    await driver.page.waitForSelector('[data-tree-path="async.go"]', {
                        visible: true,
                    })
                ).click()
                await driver.assertWindowLocation(
                    '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d/-/blob/async.go'
                )
            })

            test('expands directory on row click (no navigation)', async () => {
                await driver.page.goto(
                    sourcegraphBaseUrl + '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d'
                )
                await driver.page.waitForSelector('.tree__row-icon', { visible: true })
                await driver.page.click('.tree__row-icon')
                await driver.page.waitForSelector('.tree__row--selected [data-tree-path="websocket"]', {
                    visible: true,
                })
                await driver.page.waitForSelector('.tree__row--expanded [data-tree-path="websocket"]', {
                    visible: true,
                })
                await driver.assertWindowLocation(
                    '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d'
                )
            })

            test('does navigation on directory row click', async () => {
                await driver.page.goto(
                    sourcegraphBaseUrl + '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d'
                )
                await driver.page.waitForSelector('.tree__row-label', { visible: true })
                await driver.page.click('.tree__row-label')
                await driver.page.waitForSelector('.tree__row--selected [data-tree-path="websocket"]', {
                    visible: true,
                })
                await driver.page.waitForSelector('.tree__row--expanded [data-tree-path="websocket"]', {
                    visible: true,
                })
                await driver.assertWindowLocation(
                    '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d/-/tree/websocket'
                )
            })

            test('selects the current file', async () => {
                await driver.page.goto(
                    sourcegraphBaseUrl +
                        '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d/-/blob/async.go'
                )
                await driver.page.waitForSelector('.tree__row--active [data-tree-path="async.go"]', {
                    visible: true,
                })
            })

            test('shows partial tree when opening directory', async () => {
                await driver.page.goto(
                    sourcegraphBaseUrl +
                        '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d/-/tree/websocket'
                )
                await driver.page.waitForSelector('.tree__row', { visible: true })
                expect(await driver.page.evaluate(() => document.querySelectorAll('.tree__row').length)).toEqual(1)
            })

            test('responds to keyboard shortcuts', async () => {
                const assertNumberRowsExpanded = async (expectedCount: number): Promise<void> => {
                    expect(
                        await driver.page.evaluate(() => document.querySelectorAll('.tree__row--expanded').length)
                    ).toEqual(expectedCount)
                }
                await driver.page.goto(
                    sourcegraphBaseUrl +
                        '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/.travis.yml'
                )
                await driver.page.waitForSelector('.tree__row', { visible: true }) // waitForSelector for tree to render

                await driver.page.click('.test-repo-revision-sidebar .tree')
                await driver.page.keyboard.press('ArrowUp') // arrow up to 'diff' directory
                await driver.page.waitForSelector('.tree__row--selected [data-tree-path="diff"]', { visible: true })
                await driver.page.keyboard.press('ArrowRight') // arrow right (expand 'diff' directory)
                await driver.page.waitForSelector('.tree__row--selected [data-tree-path="diff"]', { visible: true })
                await driver.page.waitForSelector('.tree__row--expanded [data-tree-path="diff"]', { visible: true })
                await driver.page.waitForSelector('.tree__row [data-tree-path="diff/testdata"]', { visible: true })
                await driver.page.keyboard.press('ArrowRight') // arrow right (move to nested 'diff/testdata' directory)
                await driver.page.waitForSelector('.tree__row--selected [data-tree-path="diff/testdata"]', {
                    visible: true,
                })
                await assertNumberRowsExpanded(1) // only `diff` directory is expanded, though `diff/testdata` is expanded

                await driver.page.keyboard.press('ArrowRight') // arrow right (expand 'diff/testdata' directory)
                await driver.page.waitForSelector('.tree__row--selected [data-tree-path="diff/testdata"]', {
                    visible: true,
                })
                await driver.page.waitForSelector('.tree__row--expanded [data-tree-path="diff/testdata"]', {
                    visible: true,
                })
                await assertNumberRowsExpanded(2) // `diff` and `diff/testdata` directories expanded

                await driver.page.waitForSelector('.tree__row [data-tree-path="diff/testdata/empty.diff"]', {
                    visible: true,
                })
                // select some file nested under `diff/testdata`
                await driver.page.keyboard.press('ArrowDown') // arrow down
                await driver.page.keyboard.press('ArrowDown') // arrow down
                await driver.page.keyboard.press('ArrowDown') // arrow down
                await driver.page.keyboard.press('ArrowDown') // arrow down
                await driver.page.waitForSelector(
                    '.tree__row--selected [data-tree-path="diff/testdata/empty_orig.diff"]',
                    {
                        visible: true,
                    }
                )

                await driver.page.keyboard.press('ArrowLeft') // arrow left (navigate immediately up to parent directory `diff/testdata`)
                await driver.page.waitForSelector('.tree__row--selected [data-tree-path="diff/testdata"]', {
                    visible: true,
                })
                await assertNumberRowsExpanded(2) // `diff` and `diff/testdata` directories expanded

                await driver.page.keyboard.press('ArrowLeft') // arrow left
                await driver.page.waitForSelector('.tree__row--selected [data-tree-path="diff/testdata"]', {
                    visible: true,
                }) // `diff/testdata` still selected
                await assertNumberRowsExpanded(1) // only `diff` directory expanded
            })
        })
        describe('symbol sidebar', () => {
            const listSymbolsTests = [
                {
                    name: 'lists symbols in file for Go',
                    filePath:
                        '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/cmd/go-diff/go-diff.go',
                    symbolNames: ['main', 'stdin', 'diffPath', 'fileIdx', 'main'],
                    symbolTypes: ['package', 'constant', 'variable', 'variable', 'function'],
                },
                {
                    name: 'lists symbols in another file for Go',
                    filePath:
                        '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/diff.go',
                    symbolNames: [
                        'diff',
                        'Stat',
                        'Stat',
                        'hunkPrefix',
                        'hunkHeader',
                        'diffTimeParseLayout',
                        'diffTimeFormatLayout',
                        'add',
                    ],
                    symbolTypes: [
                        'package',
                        'function',
                        'function',
                        'variable',
                        'constant',
                        'constant',
                        'constant',
                        'function',
                    ],
                },
                {
                    name: 'lists symbols in file for Python',
                    filePath:
                        '/github.com/sourcegraph/appdash@ebfcffb1b5c00031ce797183546746715a3cfe87/-/blob/python/appdash/sockcollector.py',
                    symbolNames: [
                        'RemoteCollector',
                        'sock',
                        '_debug',
                        '__init__',
                        '_log',
                        'connect',
                        'collect',
                        'close',
                    ],
                    symbolTypes: ['class', 'variable', 'variable', 'field', 'field', 'field', 'field', 'field'],
                },
                {
                    name: 'lists symbols in file for TypeScript',
                    filePath:
                        '/github.com/sourcegraph/sourcegraph-typescript@a7b7a61e31af76dad3543adec359fa68737a58a1/-/blob/server/src/cancellation.ts',
                    symbolNames: [
                        'createAbortError',
                        'isAbortError',
                        'throwIfCancelled',
                        'tryCancel',
                        'toAxiosCancelToken',
                        'source',
                    ],
                    symbolTypes: ['constant', 'constant', 'function', 'function', 'function', 'constant'],
                },
                {
                    name: 'lists symbols in file for Java',
                    filePath:
                        '/github.com/sourcegraph/java-langserver@03efbe9558acc532e88f5288b4e6cfa155c6f2dc/-/blob/src/main/java/com/sourcegraph/common/Config.java',
                    symbolNames: [
                        'com.sourcegraph.common',
                        'Config',
                        'LIGHTSTEP_INCLUDE_SENSITIVE',
                        'LIGHTSTEP_PROJECT',
                        'LIGHTSTEP_TOKEN',
                        'ANDROID_JAR_PATH',
                        'IGNORE_DEPENDENCY_RESOLUTION_CACHE',
                        'LSP_TIMEOUT',
                        'LANGSERVER_ROOT',
                        'LOCAL_REPOSITORY',
                        'EXECUTE_GRADLE_ORIGINAL_ROOT_PATHS',
                        'shouldExecuteGradle',
                        'PRIVATE_REPO_ID',
                        'PRIVATE_REPO_URL',
                        'PRIVATE_REPO_USERNAME',
                        'PRIVATE_REPO_PASSWORD',
                        'log',
                        'checkEnv',
                        'ConfigException',
                    ],
                    symbolTypes: [
                        'package',
                        'class',
                        'field',
                        'field',
                        'field',
                        'field',
                        'field',
                        'field',
                        'field',
                        'field',
                        'field',
                        'method',
                        'field',
                        'field',
                        'field',
                        'field',
                        'field',
                        'method',
                        'class',
                    ],
                },
            ]

            for (const symbolTest of listSymbolsTests) {
                test(symbolTest.name, async () => {
                    await driver.page.goto(sourcegraphBaseUrl + symbolTest.filePath)

                    await (await driver.page.waitForSelector('[data-test-tab="symbols"]')).click()

                    await driver.page.waitForSelector('.test-symbol-name', { visible: true })

                    const symbolNames = await driver.page.evaluate(() =>
                        [...document.querySelectorAll('.test-symbol-name')].map(name => name.textContent || '')
                    )
                    const symbolTypes = await driver.page.evaluate(() =>
                        [...document.querySelectorAll('.test-symbol-icon')].map(
                            icon => icon.getAttribute('data-tooltip') || ''
                        )
                    )

                    expect(sortBy(symbolNames)).toEqual(sortBy(symbolTest.symbolNames))
                    expect(sortBy(symbolTypes)).toEqual(sortBy(symbolTest.symbolTypes))
                })
            }

            const navigateToSymbolTests = [
                {
                    name: 'navigates to file on symbol click for Go',
                    repoPath: '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d',
                    filePath: '/tree/cmd',
                    symbolPath: '/blob/cmd/go-diff/go-diff.go#L19:2-19:10',
                },
                {
                    name: 'navigates to file on symbol click for Java',
                    repoPath: '/github.com/sourcegraph/java-langserver@03efbe9558acc532e88f5288b4e6cfa155c6f2dc',
                    filePath: '/tree/src/main/java/com/sourcegraph/common',
                    symbolPath: '/blob/src/main/java/com/sourcegraph/common/Config.java#L14:20-14:26',
                    skip: true,
                },
                {
                    name:
                        'displays valid symbols at different file depths for Go (./examples/cmd/webapp-opentracing/main.go.go)',
                    repoPath: '/github.com/sourcegraph/appdash@ebfcffb1b5c00031ce797183546746715a3cfe87',
                    filePath: '/tree/examples',
                    symbolPath: '/blob/examples/cmd/webapp-opentracing/main.go#L26:6-26:10',
                    skip: true,
                },
                {
                    name: 'displays valid symbols at different file depths for Go (./sqltrace/sql.go)',
                    repoPath: '/github.com/sourcegraph/appdash@ebfcffb1b5c00031ce797183546746715a3cfe87',
                    filePath: '/tree/sqltrace',
                    symbolPath: '/blob/sqltrace/sql.go#L14:2-14:5',
                    skip: true,
                },
            ]

            for (const navigationTest of navigateToSymbolTests) {
                const testFunc = navigationTest.skip ? test.skip : test
                testFunc(navigationTest.name, async () => {
                    const repoBaseURL = sourcegraphBaseUrl + navigationTest.repoPath + '/-'

                    await driver.page.goto(repoBaseURL + navigationTest.filePath)

                    await (await driver.page.waitForSelector('[data-test-tab="symbols"]')).click()

                    await driver.page.waitForSelector('.test-symbol-name', { visible: true })

                    await (
                        await driver.page.waitForSelector(`.test-symbol-link[href*="${navigationTest.symbolPath}"]`, {
                            visible: true,
                        })
                    ).click()
                    await driver.assertWindowLocation(repoBaseURL + navigationTest.symbolPath, true)
                })
            }

            const highlightSymbolTests = [
                {
                    name: 'highlights correct line for Go',
                    filePath:
                        '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/diff.go',
                    index: 5,
                    line: 65,
                },
                {
                    name: 'highlights correct line for TypeScript',
                    filePath:
                        '/github.com/sourcegraph/sourcegraph-typescript@a7b7a61e31af76dad3543adec359fa68737a58a1/-/blob/server/src/cancellation.ts',
                    index: 2,
                    line: 17,
                },
            ]

            for (const { name, filePath, index, line } of highlightSymbolTests) {
                test(name, async () => {
                    await driver.page.goto(sourcegraphBaseUrl + filePath)
                    await driver.page.waitForSelector('[data-test-tab="symbols"]')
                    await driver.page.click('[data-test-tab="symbols"]')
                    await driver.page.waitForSelector('.test-symbol-name', { visible: true })
                    await driver.page.click(`.filtered-connection__nodes li:nth-child(${index + 1}) a`)

                    await driver.page.waitForSelector('.test-blob .selected .line')
                    const selectedLineNumber = await driver.page.evaluate(() => {
                        const element = document.querySelector<HTMLElement>('.test-blob .selected .line')
                        return element?.dataset.line && parseInt(element.dataset.line, 10)
                    })

                    expect(selectedLineNumber).toEqual(line)
                })
            }
        })

        describe('directory page', () => {
            it('shows a row for each file in the directory', async () => {
                await driver.page.goto(
                    sourcegraphBaseUrl + '/github.com/gorilla/securecookie@e59506cc896acb7f7bf732d4fdf5e25f7ccd8983'
                )
                await driver.page.waitForSelector('.test-tree-entries', { visible: true })
                await retry(async () =>
                    assert.equal(
                        await driver.page.evaluate(
                            () => document.querySelectorAll('.test-tree-entry-directory').length
                        ),
                        1
                    )
                )
                await retry(async () =>
                    assert.equal(
                        await driver.page.evaluate(() => document.querySelectorAll('.test-tree-entry-file').length),
                        7
                    )
                )
            })

            test('shows commit information on a row', async () => {
                await driver.page.goto(
                    sourcegraphBaseUrl + '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d',
                    {
                        waitUntil: 'domcontentloaded',
                    }
                )
                await driver.page.waitForSelector('.test-tree-page-no-recent-commits')
                await driver.page.click('.test-tree-page-show-all-commits')
                await driver.page.waitForSelector('.git-commit-node__message', { visible: true })
                await retry(async () =>
                    expect(
                        await driver.page.evaluate(
                            () => document.querySelectorAll('.git-commit-node__message')[3].textContent
                        )
                    ).toContain('Add support for new/removed binary files.')
                )
                await retry(async () =>
                    expect(
                        await driver.page.evaluate(() =>
                            document.querySelectorAll('.git-commit-node-byline')[3].textContent!.trim()
                        )
                    ).toContain('Dmitri Shuralyov')
                )
                await retry(async () =>
                    expect(
                        await driver.page.evaluate(
                            () => document.querySelectorAll('.git-commit-node__oid')[3].textContent
                        )
                    ).toEqual('2083912')
                )
            })

            it('navigates when clicking on a row', async () => {
                await driver.page.goto(
                    sourcegraphBaseUrl + '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d'
                )
                // click on directory
                await driver.page.waitForSelector('.tree-entry', { visible: true })
                await driver.page.click('.tree-entry')
                await driver.assertWindowLocation(
                    '/github.com/sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d/-/tree/websocket'
                )
            })
        })

        describe('revision resolution', () => {
            test('shows clone in progress interstitial page', async () => {
                await driver.page.goto(sourcegraphBaseUrl + '/github.com/sourcegraphtest/AlwaysCloningTest')
                await driver.page.waitForSelector('.hero-page__subtitle', { visible: true })
                await retry(async () =>
                    expect(
                        await driver.page.evaluate(() => document.querySelector('.hero-page__subtitle')!.textContent)
                    ).toEqual('Cloning in progress')
                )
            })

            test('resolves default branch when unspecified', async () => {
                await driver.page.goto(sourcegraphBaseUrl + '/github.com/sourcegraph/go-diff/-/blob/diff/diff.go')
                await driver.page.waitForSelector('#repo-revision-popover', { visible: true })
                await retry(async () => {
                    expect(
                        await driver.page.evaluate(() => document.querySelector('.test-revision')!.textContent!.trim())
                    ).toEqual('master')
                })
                // Verify file contents are loaded.
                await driver.page.waitForSelector(blobTableSelector)
            })

            test('updates revision with switcher', async () => {
                await driver.page.goto(sourcegraphBaseUrl + '/github.com/sourcegraph/go-diff/-/blob/diff/diff.go')
                // Open revision switcher
                await driver.page.waitForSelector('#repo-revision-popover', { visible: true })
                await driver.page.click('#repo-revision-popover')
                // Click "Tags" tab
                await driver.page.click('.revisions-popover .tab-bar__tab:nth-child(2)')
                await driver.page.waitForSelector('a.git-ref-node[href*="0.5.0"]', { visible: true })
                await driver.page.click('a.git-ref-node[href*="0.5.0"]')
                await driver.assertWindowLocation('/github.com/sourcegraph/go-diff@v0.5.0/-/blob/diff/diff.go')
            })
        })

        describe('hovers', () => {
            describe('Blob', () => {
                test('gets displayed and updates URL when clicking on a token', async () => {
                    await driver.page.goto(
                        sourcegraphBaseUrl +
                            '/github.com/gorilla/mux@15a353a636720571d19e37b34a14499c3afa9991/-/blob/mux.go'
                    )
                    await driver.page.waitForSelector(blobTableSelector)
                    await clickToken(24, 5)
                    await driver.assertWindowLocation(
                        '/github.com/gorilla/mux@15a353a636720571d19e37b34a14499c3afa9991/-/blob/mux.go#L24:19'
                    )
                    await getHoverContents() // verify there is a hover
                    await percySnapshot(driver.page, 'Code intel hover tooltip')
                })

                test('gets displayed when navigating to a URL with a token position', async () => {
                    await driver.page.goto(
                        sourcegraphBaseUrl +
                            '/github.com/gorilla/mux@15a353a636720571d19e37b34a14499c3afa9991/-/blob/mux.go#L151:23'
                    )
                    await assertHoverContentContains(
                        'ErrMethodMismatch is returned when the method in the request does not match'
                    )
                })

                describe('jump to definition', () => {
                    test('noops when on the definition', async () => {
                        await driver.page.goto(
                            sourcegraphBaseUrl +
                                '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/parse.go#L29:6'
                        )
                        await clickHoverJ2D()
                        await driver.assertWindowLocation(
                            '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/parse.go#L29:6'
                        )
                    })

                    test('does navigation (same repo, same file)', async () => {
                        await driver.page.goto(
                            sourcegraphBaseUrl +
                                '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/parse.go#L25:10'
                        )
                        await clickHoverJ2D()
                        await driver.assertWindowLocation(
                            '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/parse.go#L29:6'
                        )
                    })

                    test('does navigation (same repo, different file)', async () => {
                        await driver.page.goto(
                            sourcegraphBaseUrl +
                                '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/print.go#L13:31'
                        )
                        await clickHoverJ2D()
                        await driver.assertWindowLocation(
                            '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/diff.pb.go#L38:6'
                        )
                        // Verify file tree is highlighting the new path.
                        await driver.page.waitForSelector('.tree__row--active [data-tree-path="diff/diff.pb.go"]', {
                            visible: true,
                        })
                    })

                    // basic code intel doesn't support cross-repo jump-to-definition yet.
                    // If this test gets re-enabled `sourcegraph/vcsstore` and
                    // `sourcegraph/go-vcs` need to be cloned.
                    test.skip('does navigation (external repo)', async () => {
                        await driver.page.goto(
                            sourcegraphBaseUrl +
                                '/github.com/sourcegraph/vcsstore@267289226b15e5b03adedc9746317455be96e44c/-/blob/server/diff.go#L27:30'
                        )
                        await clickHoverJ2D()
                        await driver.assertWindowLocation(
                            '/github.com/sourcegraph/go-vcs@aa7c38442c17a3387b8a21f566788d8555afedd0/-/blob/vcs/repository.go#L103:6'
                        )
                    })
                })

                describe('find references', () => {
                    test('opens widget and fetches local references', async function () {
                        this.timeout(120000)

                        await driver.page.goto(
                            sourcegraphBaseUrl +
                                '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/parse.go#L29:6'
                        )
                        await clickHoverFindReferences()
                        await driver.assertWindowLocation(
                            '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/parse.go#L29:6&tab=references'
                        )

                        await driver.assertNonemptyLocalRefs()

                        // verify the appropriate # of references are fetched
                        await driver.page.waitForSelector('.panel__tabs-content .file-match-children', {
                            visible: true,
                        })
                        await retry(async () =>
                            expect(
                                await driver.page.evaluate(
                                    () =>
                                        document.querySelectorAll('.panel__tabs-content .file-match-children__item')
                                            .length
                                )
                            ).toEqual(
                                // Basic code intel finds 8 references with some overlapping context, resulting in 4 hunks.
                                4
                            )
                        )

                        // verify all the matches highlight a `MultiFileDiffReader` token
                        await driver.assertAllHighlightedTokens('MultiFileDiffReader')
                    })

                    // TODO unskip this once basic-code-intel looks for external
                    // references even when local references are found.
                    test.skip('opens widget and fetches external references', async () => {
                        await driver.page.goto(
                            sourcegraphBaseUrl +
                                '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/parse.go#L32:16&tab=references'
                        )

                        // verify some external refs are fetched (we cannot assert how many, but we can check that the matched results
                        // look like they're for the appropriate token)
                        await driver.assertNonemptyExternalRefs()

                        // verify all the matches highlight a `Reader` token
                        await driver.assertAllHighlightedTokens('Reader')
                    })
                })
            })
        })

        describe.skip('godoc.org "Uses" links', () => {
            test('resolves standard library function', async () => {
                // https://godoc.org/bytes#Compare
                await driver.page.goto(sourcegraphBaseUrl + '/-/godoc/refs?def=Compare&pkg=bytes&repo=')
                await driver.assertWindowLocationPrefix('/github.com/golang/go/-/blob/src/bytes/bytes_decl.go')
                await driver.assertStickyHighlightedToken('Compare')
                await driver.assertNonemptyLocalRefs()
                await driver.assertAllHighlightedTokens('Compare')
            })

            test('resolves standard library function (from stdlib repo)', async () => {
                // https://godoc.org/github.com/golang/go/src/bytes#Compare
                await driver.page.goto(
                    sourcegraphBaseUrl +
                        '/-/godoc/refs?def=Compare&pkg=github.com%2Fgolang%2Fgo%2Fsrc%2Fbytes&repo=github.com%2Fgolang%2Fgo'
                )
                await driver.assertWindowLocationPrefix('/github.com/golang/go/-/blob/src/bytes/bytes_decl.go')
                await driver.assertStickyHighlightedToken('Compare')
                await driver.assertNonemptyLocalRefs()
                await driver.assertAllHighlightedTokens('Compare')
            })

            test('resolves external package function (from gorilla/mux)', async () => {
                // https://godoc.org/github.com/gorilla/mux#Router
                await driver.page.goto(
                    sourcegraphBaseUrl +
                        '/-/godoc/refs?def=Router&pkg=github.com%2Fgorilla%2Fmux&repo=github.com%2Fgorilla%2Fmux'
                )
                await driver.assertWindowLocationPrefix('/github.com/gorilla/mux/-/blob/mux.go')
                await driver.assertStickyHighlightedToken('Router')
                await driver.assertNonemptyLocalRefs()
                await driver.assertAllHighlightedTokens('Router')
            })
        })

        describe('external code host links', () => {
            test('on repo navbar ("View on GitHub")', async () => {
                await driver.page.goto(
                    sourcegraphBaseUrl +
                        '/github.com/sourcegraph/go-diff@3f415a150aec0685cb81b73cc201e762e075006d/-/blob/diff/parse.go#L19',
                    { waitUntil: 'domcontentloaded' }
                )
                await driver.page.waitForSelector('.nav-link[href*="https://github"]', { visible: true })
                await retry(async () =>
                    expect(
                        await driver.page.evaluate(
                            () =>
                                (document.querySelector('.nav-link[href*="https://github"]') as HTMLAnchorElement).href
                        )
                    ).toEqual(
                        'https://github.com/sourcegraph/go-diff/blob/3f415a150aec0685cb81b73cc201e762e075006d/diff/parse.go#L19'
                    )
                )
            })
        })
    })

    describe('Search component', () => {
        test('can execute search with search operators', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/github.com/sourcegraph/go-diff')

            const operators: { [key: string]: string } = {
                repo: '^github.com/sourcegraph/go-diff$',
                count: '1000',
                type: 'file',
                file: '.go',
                '-file': '.md',
            }

            const operatorsQuery = Object.keys(operators)
                .map(operator => `${operator}:${operators[operator]}`)
                .join('+')

            await driver.page.goto(`${sourcegraphBaseUrl}/search?q=diff+${operatorsQuery}&patternType=regexp`)
            await driver.page.waitForSelector('.test-search-results-stats', { visible: true })
            await retry(async () => {
                const label = await driver.page.evaluate(
                    () => document.querySelector('.test-search-results-stats')!.textContent || ''
                )
                expect(label.includes('results')).toEqual(true)
            })
            await driver.page.waitForSelector('.test-file-match-children-item', { visible: true })
        })

        test('renders results for sourcegraph/go-diff (no search group)', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/github.com/sourcegraph/go-diff')
            await driver.page.goto(
                sourcegraphBaseUrl +
                    '/search?q=diff+repo:sourcegraph/go-diff%403f415a150aec0685cb81b73cc201e762e075006d+type:file&patternType=regexp'
            )
            await driver.page.waitForSelector('.test-search-results-stats', { visible: true })
            await retry(async () => {
                const label = await driver.page.evaluate(
                    () => document.querySelector('.test-search-results-stats')!.textContent || ''
                )
                expect(label.includes('results')).toEqual(true)
            })

            const firstFileMatchHref = await driver.page.$eval(
                '.test-file-match-children-item',
                a => (a as HTMLAnchorElement).href
            )

            // navigate to result on click
            await driver.page.click('.test-file-match-children-item')

            await retry(async () => {
                expect(await driver.page.evaluate(() => window.location.href)).toEqual(firstFileMatchHref)
            })
        })

        describe('multiple revisions per repository', () => {
            let previousExperimentalFeatures: any
            before(async () => {
                await driver.setConfig(['experimentalFeatures'], previous => {
                    previousExperimentalFeatures = previous?.value
                    return { searchMultipleRevisionsPerRepository: true }
                })
                // Wait for configuration to be applied.
                await new Promise(resolve => setTimeout(resolve, 6000))
            })
            after(async () => {
                await driver.setConfig(['experimentalFeatures'], () => previousExperimentalFeatures)
            })

            test('searches', async () => {
                await driver.page.goto(
                    sourcegraphBaseUrl +
                        '/search?q=repo:sourcegraph/go-diff%24%40master:print-options:*refs/heads/+func+NewHunksReader&patternType=regexp'
                )
                await driver.page.waitForSelector('.test-search-results-stats', { visible: true })
                await retry(async () => {
                    const label = await driver.page.evaluate(
                        () => document.querySelector('.test-search-results-stats')!.textContent || ''
                    )
                    expect(label.includes('results')).toEqual(true)
                })

                const fileMatchHrefs = (
                    await driver.page.$$eval('.test-file-match-children-item', as =>
                        as.map(a => (a as HTMLAnchorElement).pathname)
                    )
                ).sort()

                // Only check for specific branches, so that the test doesn't break when new
                // branches are added (it's an active repository).
                const checkBranches = [
                    'master',
                    'print-options',

                    // These next 2 branches are included because of the *refs/heads/ in the query.
                    // If they are ever deleted from the actual live repository, replace them with
                    // any other branches that still exist.
                    'test-already-exist-pr',
                    'bug-fix-wip',
                ].sort()
                expect(
                    fileMatchHrefs.filter(href => checkBranches.some(branch => href.includes(`@${branch}/`)))
                ).toEqual(checkBranches.map(branch => `/github.com/sourcegraph/go-diff@${branch}/-/blob/diff/parse.go`))
            })
        })

        test('accepts query for sourcegraph/jsonrpc2', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/search')

            // Update the input value
            await driver.page.waitForSelector('#monaco-query-input', { visible: true })
            await driver.page.keyboard.type('test repo:sourcegraph/jsonrpc2@c6c7b9aa99fb76ee5460ccd3912ba35d419d493d')

            // TODO: test search scopes

            // Submit the search
            await driver.page.click('.search-button')

            await driver.page.waitForSelector('.test-search-results-stats', { visible: true })
            await retry(async () => {
                const label = await driver.page.evaluate(
                    () => document.querySelector('.test-search-results-stats')!.textContent || ''
                )
                const match = /(\d+) results?/.exec(label)
                if (!match) {
                    throw new Error(
                        `.test-search-results-stats textContent did not match regex '(\\d+) results': '${label}'`
                    )
                }
                const numberOfResults = parseInt(match[1], 10)
                expect(numberOfResults).toBeGreaterThan(0)
            })
        })

        test('redirects to a URL with &patternType=regexp if no patternType in URL', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/search?q=test')
            await driver.assertWindowLocation('/search?q=test&patternType=regexp')
        })

        test('regexp toggle appears and updates patternType query parameter when clicked', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/search?q=test&patternType=literal')
            // Wait for monaco query input to load to avoid race condition with the intermediate input
            await driver.page.waitForSelector('#monaco-query-input')
            await driver.page.waitForSelector('.test-regexp-toggle')
            await driver.page.click('.test-regexp-toggle')
            await driver.page.goto(sourcegraphBaseUrl + '/search?q=test&patternType=regexp')
            // Wait for monaco query input to load to avoid race condition with the intermediate input
            await driver.page.waitForSelector('#monaco-query-input')
            await driver.page.waitForSelector('.test-regexp-toggle')
            await driver.page.click('.test-regexp-toggle')
            await driver.page.goto(sourcegraphBaseUrl + '/search?q=test&patternType=literal')
        })
    })

    describe.skip('Search pattern type setting', () => {
        test('Search pattern type setting correctly sets default pattern type', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/users/test/settings')
            await driver.replaceText({
                selector: '.test-settings-file .monaco-editor',
                newText: JSON.stringify({
                    'search.defaultPatternType': 'regexp',
                }),
                selectMethod: 'keyboard',
            })
            await driver.page.click('.test-settings-file .test-save-toolbar-save')

            await driver.page.goto(sourcegraphBaseUrl + '/search')
            await driver.page.waitForSelector('#monaco-query-input', { visible: true })
            await driver.page.waitForSelector('.test-regexp-toggle', { visible: true })

            const activeToggle = await driver.page.evaluate(
                () => document.querySelectorAll('.test-regexp-toggle--active').length
            )
            expect(activeToggle).toEqual(1)
            await driver.page.keyboard.type('test')
            await driver.page.click('.search-button')
            await driver.assertWindowLocation('/search?q=test&patternType=regexp')
        })
    })

    describe('Search result type tabs', () => {
        test('Search results type tabs appear', async () => {
            await driver.page.goto(
                sourcegraphBaseUrl + '/search?q=repo:%5Egithub.com/gorilla/mux%24&patternType=regexp'
            )
            await driver.page.waitForSelector('.test-search-result-type-tabs', { visible: true })
            await driver.page.waitForSelector('.test-search-result-tab--active', { visible: true })
            const tabs = await driver.page.evaluate(() =>
                [...document.querySelectorAll('.test-search-result-tab')].map(tab => tab.textContent)
            )

            expect(tabs.length).toEqual(6)
            expect(tabs).toStrictEqual(['Code', 'Diffs', 'Commits', 'Symbols', 'Repositories', 'Filenames'])

            const activeTab = await driver.page.evaluate(
                () => document.querySelectorAll('.test-search-result-tab--active').length
            )
            expect(activeTab).toEqual(1)

            const label = await driver.page.evaluate(
                () => document.querySelector('.test-search-result-tab--active')!.textContent || ''
            )
            expect(label).toEqual('Code')
        })

        test.skip('Clicking search results tabs updates query and URL', async () => {
            for (const searchType of ['diff', 'commit', 'symbol', 'repo']) {
                await driver.page.waitForSelector(`.test-search-result-tab-${searchType}`)
                await driver.page.click(`.test-search-result-tab-${searchType}`)
                await driver.assertWindowLocation(
                    `/search?q=repo:%5Egithub.com/gorilla/mux%24+type:${searchType}&patternType=regexp`
                )
            }
        })
    })

    describe('Saved searches', () => {
        test('Save search from search results page', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/search?q=test')
            await driver.page.waitForSelector('.test-save-search-link', { visible: true })
            await driver.page.click('.test-save-search-link')
            await driver.page.waitForSelector('.test-saved-search-modal')
            await driver.page.waitForSelector('.test-saved-search-modal-save-button')
            await driver.page.click('.test-saved-search-modal-save-button')
            await driver.assertWindowLocation('/users/test/searches/add?query=test&patternType=regexp')

            await driver.page.waitForSelector('.test-saved-search-form-input-description', { visible: true })
            await driver.page.click('.test-saved-search-form-input-description')
            await driver.page.keyboard.type('test query')
            await driver.page.waitForSelector('.test-saved-search-form-submit-button', { visible: true })
            await driver.page.click('.test-saved-search-form-submit-button')
            await driver.assertWindowLocation('/users/test/searches')

            const nodes = await driver.page.evaluate(
                () => document.querySelectorAll('.test-saved-search-list-page-row').length
            )
            expect(nodes).toEqual(1)

            expect(
                await driver.page.evaluate(
                    () => document.querySelector('.test-saved-search-list-page-row-title')!.textContent
                )
            ).toEqual('test query')
        })
        test('Delete saved search', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/users/test/searches')
            await driver.page.waitForSelector('.test-delete-saved-search-button', { visible: true })
            driver.page.on('dialog', async dialog => {
                await dialog.accept()
            })
            await driver.page.click('.test-delete-saved-search-button')
            await driver.page.waitFor(() => !document.querySelector('.test-saved-search-list-page-row'))
            const nodes = await driver.page.evaluate(
                () => document.querySelectorAll('.test-saved-search-list-page-row').length
            )
            expect(nodes).toEqual(0)
        })
        test('Save search from saved searches page', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/users/test/searches')
            await driver.page.waitForSelector('.test-add-saved-search-button', { visible: true })
            await driver.page.click('.test-add-saved-search-button')
            await driver.assertWindowLocation('/users/test/searches/add')

            await driver.page.waitForSelector('.test-saved-search-form-input-description', { visible: true })
            await driver.page.click('.test-saved-search-form-input-description')
            await driver.page.keyboard.type('test query 2')

            await driver.page.waitForSelector('.test-saved-search-form-input-query', { visible: true })
            await driver.page.click('.test-saved-search-form-input-query')
            await driver.page.keyboard.type('test patternType:literal')

            await driver.page.waitForSelector('.test-saved-search-form-submit-button', { visible: true })
            await driver.page.click('.test-saved-search-form-submit-button')
            await driver.assertWindowLocation('/users/test/searches')

            const nodes = await driver.page.evaluate(
                () => document.querySelectorAll('.test-saved-search-list-page-row').length
            )
            expect(nodes).toEqual(1)

            expect(
                await driver.page.evaluate(
                    () => document.querySelector('.test-saved-search-list-page-row-title')!.textContent
                )
            ).toEqual('test query 2')
        })
        test('Edit saved search', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/users/test/searches')
            await driver.page.waitForSelector('.test-edit-saved-search-button', { visible: true })
            await driver.page.click('.test-edit-saved-search-button')

            await driver.page.waitForSelector('.test-saved-search-form-input-description', { visible: true })
            await driver.page.click('.test-saved-search-form-input-description')
            await driver.page.keyboard.type(' edited')

            await driver.page.waitForSelector('.test-saved-search-form-submit-button', { visible: true })
            await driver.page.click('.test-saved-search-form-submit-button')
            await driver.page.goto(sourcegraphBaseUrl + '/users/test/searches')
            await driver.page.waitForSelector('.test-saved-search-list-page-row-title')

            expect(
                await driver.page.evaluate(
                    () => document.querySelector('.test-saved-search-list-page-row-title')!.textContent
                )
            ).toEqual('test query 2 edited')
        })
    })

    describe('Search statistics', () => {
        beforeEach(async () => {
            await driver.setUserSettings<Settings>({ experimentalFeatures: { searchStats: true } })
        })
        afterEach(async () => {
            await driver.resetUserSettings()
        })

        // This is a substring that appears in the sourcegraph/go-diff repository, which is present
        // in the external service added for the e2e test. It is OK if it starts to appear in other
        // repositories (such as sourcegraph/sourcegraph now that it's mentioned here); the test
        // just checks that it is found in at least 1 Go file.
        const uniqueString = 'Incomplete-'
        const uniqueStringPostfix = 'Lines'

        test('button on search results page', async () => {
            await driver.page.goto(`${sourcegraphBaseUrl}/search?q=${uniqueString}`)
            await driver.page.waitForSelector(`a[href="/stats?q=${uniqueString}"]`)
        })

        test('page', async () => {
            await driver.page.goto(`${sourcegraphBaseUrl}/stats?q=${uniqueString}`)

            // Ensure the global navbar hides the search input (to avoid confusion with the one on
            // the stats page).
            await driver.page.waitForSelector('.global-navbar a.nav-link[href="/search"]')
            assert.strictEqual(
                await driver.page.evaluate(() => document.querySelectorAll('#monaco-query-input').length),
                0
            )

            const queryInputValue = () =>
                driver.page.evaluate(() => {
                    const input = document.querySelector<HTMLInputElement>('.test-stats-query')
                    return input ? input.value : null
                })

            // Check for a Go result (the sample repositories have Go files).
            await driver.page.waitForSelector(`a[href*="${uniqueString}+lang:go"]`)
            assert.strictEqual(await queryInputValue(), uniqueString)
            await percySnapshot(driver.page, 'Search stats')

            // Update the query and rerun the computation.
            await driver.page.type('.test-stats-query', uniqueStringPostfix) // the uniqueString is followed by 'Incomplete-Lines' in go-diff
            const wantQuery = `${uniqueString}${uniqueStringPostfix}`
            assert.strictEqual(await queryInputValue(), wantQuery)
            await driver.page.click('.test-stats-query-update')
            await driver.page.waitForSelector(`a[href*="${wantQuery}+lang:go"]`)
            assert.ok(driver.page.url().endsWith(`/stats?q=${wantQuery}`))
        })
    })
})
