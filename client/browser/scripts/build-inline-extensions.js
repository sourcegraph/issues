const path = require('path')
const shelljs = require('shelljs')
const extensionNames = require('../bundled-code-intel-extensions.json').extensions || []
const toDirectory = path.join(process.cwd(), 'build')
const temporarySourceDirectory = path.join(process.cwd(), 'code-intel-extensions')
const signale = require('signale')

// Check if code-intel-extensions has already been fetched
const codeIntelExtensionsDirectoryExists = shelljs.test('-d', temporarySourceDirectory)
if (codeIntelExtensionsDirectoryExists) {
  console.log('Found existing code-intel-extensions.')
} else {
  console.log('Did not find an existing code-intel-extensions. Running fetch-code-intel-extensions')
  shelljs.exec('yarn run fetch-code-intel-extensions')
}

// Install dependencies
shelljs.exec('yarn --cwd code-intel-extensions install')

// Build individual extensions
for (const extensionName of extensionNames) {
  const extensionDirectory = `${temporarySourceDirectory}/extensions/${extensionName}`
  if (!shelljs.test('-d', extensionDirectory)) {
    console.error(`Code intel extension "${extensionName}" was not found for bundling.`)
    console.error(`Expected extension directory: ${extensionDirectory}`)
    process.exit(1)
  }
  shelljs.exec(`yarn --cwd ${temporarySourceDirectory}/extensions/${extensionName} run build`)
}

for (const extensionName of extensionNames) {
  // Copy extension manifest (package.json) and bundle (extension.js)
  shelljs.mkdir('-p', `${toDirectory}/extensions/${extensionName}`)

  shelljs.cp(
    `${temporarySourceDirectory}/extensions/${extensionName}/dist/extension.js`,
    `${toDirectory}/extensions/${extensionName}`
  )
  shelljs.cp(
    `${temporarySourceDirectory}/extensions/${extensionName}/package.json`,
    `${toDirectory}/extensions/${extensionName}`
  )
}

signale.success('Done building Sourcegraph extensions')
