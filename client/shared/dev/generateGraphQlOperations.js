// @ts-check

const { exec } = require('child_process')
const path = require('path')

const { generate } = require('@graphql-codegen/cli')
const globby = require('globby')

const ROOT_FOLDER = path.resolve(__dirname, '../../../')

const WEB_FOLDER = path.resolve(ROOT_FOLDER, './client/web')
const BROWSER_FOLDER = path.resolve(ROOT_FOLDER, './client/browser')
const SHARED_FOLDER = path.resolve(ROOT_FOLDER, './client/shared')
const SCHEMA_PATH = path.join(ROOT_FOLDER, './cmd/frontend/graphqlbackend/*.graphql')

const SHARED_DOCUMENTS_GLOB = [
  `${SHARED_FOLDER}/src/**/*.{ts,tsx}`,
  `!${SHARED_FOLDER}/src/testing/**/*.*`,
  `!${SHARED_FOLDER}/src/graphql/schema.ts`,
]

const WEB_DOCUMENTS_GLOB = [
  `${WEB_FOLDER}/src/**/*.{ts,tsx}`,
  `!${WEB_FOLDER}/src/regression/**/*.*`,
  `!${WEB_FOLDER}/src/end-to-end/**/*.*`,
]

const BROWSER_DOCUMENTS_GLOB = [
  `${BROWSER_FOLDER}/src/**/*.{ts,tsx}`,
  `!${BROWSER_FOLDER}/src/end-to-end/**/*.*`,
  '!**/*.d.ts',
]

// Define ALL_DOCUMENTS_GLOB as the union of the previous glob arrays.
const ALL_DOCUMENTS_GLOB = [...new Set([...SHARED_DOCUMENTS_GLOB, ...WEB_DOCUMENTS_GLOB, ...BROWSER_DOCUMENTS_GLOB])]

const plugins = [`${SHARED_FOLDER}/dev/extractGraphQlOperationCodegenPlugin.js`, 'typescript', 'typescript-operations']

/**
 * Generates TypeScript files with types for all GraphQL operations.
 */
async function generateGraphQlOperations() {
  await generate(
    {
      schema: SCHEMA_PATH,
      hooks: {
        afterOneFileWrite: 'prettier --write',
      },
      errorsOnly: true,
      config: {
        preResolveTypes: true,
        operationResultSuffix: 'Result',
        omitOperationSuffix: true,
        skipTypename: true,
        namingConvention: {
          typeNames: 'keep',
          enumValues: 'keep',
          transformUnderscore: true,
        },
        declarationKind: 'interface',
        avoidOptionals: {
          field: true,
          inputValue: false,
          object: true,
        },
        scalars: {
          DateTime: 'string',
          JSON: 'object',
          JSONValue: 'unknown',
          GitObjectID: 'string',
          JSONCString: 'string',
          PublishedValue: "boolean | 'draft'",
          BigInt: 'string',
        },
      },
      generates: {
        [path.join(BROWSER_FOLDER, './src/graphql-operations.ts')]: {
          documents: await globToFiles(BROWSER_DOCUMENTS_GLOB),
          config: {
            onlyOperationTypes: true,
            noExport: false,
            enumValues: '@sourcegraph/shared/src/graphql-operations',
            interfaceNameForOperations: 'BrowserGraphQlOperations',
          },
          plugins,
        },

        [path.join(WEB_FOLDER, './src/graphql-operations.ts')]: {
          documents: await globToFiles(WEB_DOCUMENTS_GLOB),
          config: {
            onlyOperationTypes: true,
            noExport: false,
            enumValues: '@sourcegraph/shared/src/graphql-operations',
            interfaceNameForOperations: 'WebGraphQlOperations',
          },
          plugins,
        },

        [path.join(SHARED_FOLDER, './src/graphql-operations.ts')]: {
          documents: await globToFiles(SHARED_DOCUMENTS_GLOB),
          config: {
            onlyOperationTypes: true,
            noExport: false,
            interfaceNameForOperations: 'SharedGraphQlOperations',
          },
          plugins,
        },
      },
    },
    true
  )
}

/**
 * Shells out to grep to find files that contain the gql tag. Other files are irrelevant and the
 * built-in more precise AST parsing is very slow, so we make this simplification here.
 *
 * @param {string[]} globs the globs to use.
 *
 * @returns {Promise<string[]>} List of files that may need to be included in the type generation.
 */
async function globToFiles(globs) {
  const schemaFiles = await globby(globs)
  return new Promise((resolve, reject) =>
    exec(`grep "gql\\\`" ${schemaFiles.join(' ')} -Rl`, (error, stdout) => {
      if (error) {
        reject(error)
      }
      resolve(stdout.split('\n'))
    })
  )
}

module.exports = {
  generateGraphQlOperations,
  SHARED_DOCUMENTS_GLOB,
  WEB_DOCUMENTS_GLOB,
  ALL_DOCUMENTS_GLOB,
}
