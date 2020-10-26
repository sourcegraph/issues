const shelljs = require('shelljs')
const path = require('path')
const signale = require('signale')
const bundledCodeIntelExtensionsConfig = require('../bundled-code-intel-extensions.json')
const temporarySourceDirectory = path.join(process.cwd(), 'code-intel-extensions')
shelljs.set('-e')

/**
 * Revision of the github.com/sourcegraph/code-intel-extensions repo to build
 * from.
 *
 * We freeze it at a particular revision because the XPI needs to be
 * reproducible. The revision should be updated whenever a new version of the
 * browser extension is being built for release.
 */
const codeIntelExtensionsRepoRevision = bundledCodeIntelExtensionsConfig.revision

signale.await('Fetching code-intel-extensions')

// Clean old directory before re-fetching
shelljs.rm('-rf', temporarySourceDirectory)

// Get code-intel-extensions source snapshot and build
shelljs.exec(
  `curl -OLs https://github.com/sourcegraph/code-intel-extensions/archive/${codeIntelExtensionsRepoRevision}.zip`
)

// - Clean: remove old unzipped directory in case of an interrupted process.
shelljs.rm('-rf', `code-intel-extensions-${codeIntelExtensionsRepoRevision}`)

// - Unzip it, which creates a new directory: code-intel-extensions-{rev}
shelljs.exec(`unzip -q ${codeIntelExtensionsRepoRevision}.zip`)

// - Rename directory to remove revision suffix
shelljs.mv(`code-intel-extensions-${codeIntelExtensionsRepoRevision}`, temporarySourceDirectory)

shelljs.rm(`${codeIntelExtensionsRepoRevision}.zip`)
