import { writeFile } from 'mz/fs'
import latestVersion from 'latest-version'
import signale from 'signale'
import * as semver from 'semver'
import execa from 'execa'
import * as path from 'path'

// Publish the native integration to npm

async function main(): Promise<void> {
    const name = '@sourcegraph/code-host-integration'
    // Bump version
    let version: string
    try {
        const currentVersion = await latestVersion(name)
        signale.info(`Current version is ${currentVersion}`)
        version = semver.inc(currentVersion, 'patch')!
    } catch (error) {
        if (error && error.name === 'PackageNotFoundError') {
            signale.info('Package is not released yet')
            version = '0.0.0'
        } else {
            throw error
        }
    }
    const packageJson = {
        name,
        version,
        license: 'Apache-2.0',
        repository: {
            type: 'git',
            url: 'https://github.com/sourcegraph/sourcegraph',
            directory: 'browser',
        },
    }
    signale.info(`New version is ${packageJson.version}`)
    // Write package.json
    const packagePath = path.resolve(__dirname, '..', 'build', 'integration')
    await writeFile(path.join(packagePath, 'package.json'), JSON.stringify(packageJson, null, 2))
    if (!process.env.CI) {
        signale.warn('Not running in CI, aborting')
        return
    }
    // Publish
    await execa('npm', ['publish', '--access', 'public'], { cwd: packagePath, stdio: 'inherit' })
}
main().catch(error => {
    process.exitCode = 1
    console.error(error)
})
