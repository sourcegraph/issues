const shelljs = require('shelljs')
const signale = require('signale')

/**
 * Purpose of this script: create a source code zip that can be used shared and
 * used to produce a build of the browser extensions.
 *
 * This makes it possible to provide the complete source code to the maintainers
 * of extension registries, in the cases where providing the source code is a
 * pre-requisite to getting approval for publishing.
 *
 * This script fetches a fresh copy of the repository, identified by commit ID.
 * This is done instead of using packaging the contents of the current working
 * copy repository, and the reason is to avoid including any build artifacts
 * that may be present. By pinning it to a commit, it's possible to use this
 * script across any branch and commit.
 *
 */

// Configuration
const commitId = '350764282631014ea24ccd88fda459a4b19a5669'
const includeCodeIntelExtensions = true
const rootDirectoryNameForZip = 'sourcegraph-source'

shelljs.rm('-f', 'sourcegraph.zip')
signale.await(`Downloading sourcegraph/sourcegraph at revision ${commitId}`)
shelljs.exec(
  `curl -Ls https://github.com/sourcegraph/sourcegraph/archive/${commitId}.zip -o sourcegraph-downloaded.zip`
)
shelljs.rm('-rf', `sourcegraph-${commitId}/`)
shelljs.exec('unzip -q sourcegraph-downloaded.zip')
shelljs.rm('-f', 'sourcegraph-downloaded.zip')
shelljs.mv(`sourcegraph-${commitId}`, rootDirectoryNameForZip)
signale.success('Downloaded and unzipped sourcegraph/sourcegraph repository')

if (includeCodeIntelExtensions) {
  shelljs.pushd(rootDirectoryNameForZip)
  shelljs.exec('yarn install')
  shelljs.exec('yarn --cwd browser run fetch-code-intel-extensions')
  shelljs.popd()
}

signale.await('Producing sourcegraph.zip')
shelljs.exec(`zip -qr sourcegraph.zip ${rootDirectoryNameForZip} --exclude "${rootDirectoryNameForZip}/node_modules/*"`)
shelljs.rm('-rf', rootDirectoryNameForZip)
signale.success('Done producing sourcegraph.zip')
