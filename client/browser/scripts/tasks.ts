/* eslint no-sync: warn */
import fs from 'fs'
import { omit } from 'lodash'
import path from 'path'
import shelljs from 'shelljs'
import signale from 'signale'
import utcVersion from 'utc-version'
import { Stats } from 'webpack'
import extensionInfo from '../src/browser-extension/manifest.spec.json'
import schema from '../src/browser-extension/schema.json'

/**
 * If true, add <all_urls> to the permissions in the manifest.
 * This is needed for e2e and integration tests because it is not possible to accept the
 * permission prompt with puppeteer.
 */
const EXTENSION_PERMISSIONS_ALL_URLS = Boolean(
    process.env.EXTENSION_PERMISSIONS_ALL_URLS && JSON.parse(process.env.EXTENSION_PERMISSIONS_ALL_URLS)
)

export type BuildEnvironment = 'dev' | 'prod'

type Browser = 'firefox' | 'chrome'

const BUILDS_DIR = 'build'

/*
 * Use a UTC-timestamp-based as the version string, generated at build-time.
 *
 * If enabled, the version string will depend on the timestamp when building, so
 * it will vary with every build. Uses the `utc-version` module.
 *
 * To get a reproducible build, disable this and set a version manually in
 * `manifest.spec.json`.
 */
const useUtcVersion = true

export const WEBPACK_STATS_OPTIONS: Stats.ToStringOptions = {
    all: false,
    timings: true,
    errors: true,
    warnings: true,
    colors: true,
}

function ensurePaths(): void {
    shelljs.mkdir('-p', 'build/dist')
    shelljs.mkdir('-p', 'build/bundles')
    shelljs.mkdir('-p', 'build/chrome')
    shelljs.mkdir('-p', 'build/firefox')
}

export function copyAssets(): void {
    signale.await('Copy assets')
    const directory = 'build/dist'
    shelljs.rm('-rf', directory)
    shelljs.mkdir('-p', directory)
    shelljs.cp('-R', 'assets/*', directory)
    shelljs.cp('-R', 'src/browser-extension/pages/*', directory)
    signale.success('Assets copied')
}

function copyExtensionAssets(toDirectory: string): void {
    shelljs.mkdir('-p', `${toDirectory}/js`, `${toDirectory}/css`, `${toDirectory}/img`)
    shelljs.cp('build/dist/js/*.bundle.js', `${toDirectory}/js`)
    shelljs.cp('build/dist/css/*.bundle.css', `${toDirectory}/css`)
    shelljs.cp('-R', 'build/dist/img/*', `${toDirectory}/img`)
    shelljs.cp('build/dist/*.html', toDirectory)
}

/**
 * When building with inline (bundled) Sourcegraph extensions, copy the built Sourcegraph extensions into the output.
 * They will be available as `web_accessible_resources`.
 *
 * The pre-requisite step is to first clone, build, and copy into `build/extensions`.
 */
function copyInlineExtensions(toDirectory: string): void {
    shelljs.cp('-R', 'build/extensions', toDirectory)
}

export function copyIntegrationAssets(): void {
    shelljs.mkdir('-p', 'build/integration/scripts')
    shelljs.mkdir('-p', 'build/integration/css')
    shelljs.cp('build/dist/js/phabricator.bundle.js', 'build/integration/scripts')
    shelljs.cp('build/dist/js/integration.bundle.js', 'build/integration/scripts')
    shelljs.cp('build/dist/js/extensionHostWorker.bundle.js', 'build/integration/scripts')
    shelljs.cp('build/dist/css/style.bundle.css', 'build/integration/css')
    shelljs.cp('src/native-integration/extensionHostFrame.html', 'build/integration')
    // Copy to the ui/assets directory so that these files can be served by
    // the webapp.
    shelljs.mkdir('-p', '../../ui/assets/extension')
    shelljs.cp('-r', 'build/integration/*', '../../ui/assets/extension')
}

const BROWSER_TITLES = {
    firefox: 'Firefox',
    chrome: 'Chrome',
}

const BROWSER_BUNDLE_ZIPS = {
    firefox: 'firefox-bundle.xpi',
    chrome: 'chrome-bundle.zip',
}

const BROWSER_BLOCKLIST = {
    chrome: ['applications'] as const,
    firefox: ['key'] as const,
}

function writeSchema(environment: BuildEnvironment, browser: Browser, writeDirectory: string): void {
    fs.writeFileSync(`${writeDirectory}/schema.json`, JSON.stringify(schema, null, 4))
}

const version = process.env.BROWSER_EXTENSION_VERSION || utcVersion()

const shouldBuildWithInlineExtensions = (browser: Browser): boolean => browser === 'firefox'

function writeManifest(environment: BuildEnvironment, browser: Browser, writeDirectory: string): void {
    const manifest = {
        ...omit(extensionInfo, ['dev', 'prod', ...BROWSER_BLOCKLIST[browser]]),
        ...omit(extensionInfo[environment], BROWSER_BLOCKLIST[browser]),
    }

    if (EXTENSION_PERMISSIONS_ALL_URLS) {
        manifest.permissions!.push('<all_urls>')
        signale.info('Adding <all_urls> to permissions because of env var setting')
    }

    if (browser === 'firefox') {
        manifest.permissions!.push('<all_urls>')
        delete manifest.storage
    }

    if (shouldBuildWithInlineExtensions(browser)) {
        // Add the inline extensions to web accessible resources
        manifest.web_accessible_resources = manifest.web_accessible_resources || []
        manifest.web_accessible_resources.push('extensions/*')

        // Revert the CSP to default, in order to remove the `blob` policy exception.
        delete manifest.content_security_policy
    }

    delete manifest.$schema

    if (environment === 'prod' && useUtcVersion) {
        manifest.version = version
    }

    fs.writeFileSync(`${writeDirectory}/manifest.json`, JSON.stringify(manifest, null, 4))
}

function buildForBrowser(browser: Browser): (environment: BuildEnvironment) => () => void {
    ensurePaths()
    return environment => {
        const title = BROWSER_TITLES[browser]

        const buildDirectory = path.resolve(process.cwd(), `${BUILDS_DIR}/${browser}`)

        writeManifest(environment, browser, buildDirectory)
        writeSchema(environment, browser, buildDirectory)

        return () => {
            // Allow only building for specific browser targets.
            // Useful in local dev for faster builds.
            if (process.env.TARGETS && !process.env.TARGETS.includes(browser)) {
                return
            }

            signale.await(`Building the ${title} ${environment} bundle`)

            copyExtensionAssets(buildDirectory)
            if (shouldBuildWithInlineExtensions(browser)) {
                copyInlineExtensions(buildDirectory)
            }

            const zipDestination = path.resolve(process.cwd(), `${BUILDS_DIR}/bundles/${BROWSER_BUNDLE_ZIPS[browser]}`)
            if (zipDestination) {
                shelljs.mkdir('-p', `./${BUILDS_DIR}/bundles`)
                shelljs.exec(`cd ${buildDirectory} && zip -q -r ${zipDestination} *`)
            }

            signale.success(`Done building the ${title} ${environment} bundle`)
        }
    }
}

export const buildFirefox = buildForBrowser('firefox')
export const buildChrome = buildForBrowser('chrome')
