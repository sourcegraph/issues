// @ts-check

const { generateNamespace } = require('@gql2ts/from-schema')
const { DEFAULT_OPTIONS, DEFAULT_TYPE_MAP } = require('@gql2ts/language-typescript')
const { buildSchema, introspectionFromSchema } = require('graphql')
const gulp = require('gulp')
const { compile: compileJSONSchema } = require('json-schema-to-typescript')
const { readFile, writeFile, mkdir } = require('mz/fs')
const path = require('path')
const { format, resolveConfig } = require('prettier')

const { generateGraphQlOperations } = require('./dev/generateGraphQlOperations')

const GRAPHQL_SCHEMA_PATH = path.join(__dirname, '../../cmd/frontend/graphqlbackend/schema.graphql')

/**
 * Generates the TypeScript types for the GraphQL schema.
 * These are used by older code, new code should rely on the new query-specific generated types.
 *
 * @returns {Promise<void>}
 */
async function graphQlSchema() {
  const schemaString = await readFile(GRAPHQL_SCHEMA_PATH, 'utf8')
  const schema = buildSchema(schemaString)

  const result = introspectionFromSchema(schema)

  const formatOptions = await resolveConfig(__dirname, { config: __dirname + '/../../prettier.config.js' })
  const typings =
    'export type ID = string\n' +
    'export type GitObjectID = string\n' +
    'export type DateTime = string\n' +
    'export type JSONCString = string\n' +
    '\n' +
    generateNamespace(
      '',
      result,
      {
        typeMap: {
          ...DEFAULT_TYPE_MAP,
          ID: 'ID',
          GitObjectID: 'GitObjectID',
          DateTime: 'DateTime',
          JSONCString: 'JSONCString',
        },
      },
      {
        generateNamespace: (name, interfaces) => interfaces,
        interfaceBuilder: (name, body) => `export ${DEFAULT_OPTIONS.interfaceBuilder(name, body)}`,
        enumTypeBuilder: (name, values) =>
          `export ${DEFAULT_OPTIONS.enumTypeBuilder(name, values).replace(/^const enum/, 'enum')}`,
        typeBuilder: (name, body) => `export ${DEFAULT_OPTIONS.typeBuilder(name, body)}`,
        wrapList: type => `${type}[]`,
        postProcessor: code => format(code, { ...formatOptions, parser: 'typescript' }),
      }
    )
  await writeFile(__dirname + '/src/graphql/schema.ts', typings)
}

/**
 * Generates the legacy graphql.ts types on file changes.
 */
async function watchGraphQlSchema() {
  await new Promise((resolve, reject) => {
    gulp.watch(GRAPHQL_SCHEMA_PATH, graphQlSchema).on('error', reject)
  })
}

/**
 * Generates the new query-specific types on file changes.
 */
async function watchGraphQlOperations() {
  await generateGraphQlOperations({ watch: true })
}

/**
 * Generates the new query-specific types.
 */
async function graphQlOperations() {
  await generateGraphQlOperations()
}

/**
 * Allow json-schema-ref-parser to resolve the v7 draft of JSON Schema
 * using a local copy of the spec, enabling developers to run/develop Sourcegraph offline
 */
const draftV7resolver = {
  order: 1,
  read: () => readFile(path.join(__dirname, '../../schema/json-schema-draft-07.schema.json')),
  canRead: file => file.url === 'http://json-schema.org/draft-07/schema',
}

/**
 * Generates the TypeScript types for the JSON schemas.
 *
 * @returns {Promise<void>}
 */
async function schema() {
  const outputDirectory = path.join(__dirname, '..', 'web', 'src', 'schema')
  await mkdir(outputDirectory, { recursive: true })
  const schemaDirectory = path.join(__dirname, '..', '..', 'schema')
  await Promise.all(
    ['json-schema-draft-07', 'settings', 'site'].map(async file => {
      let schema = await readFile(path.join(schemaDirectory, `${file}.schema.json`), 'utf8')
      // HACK: Rewrite absolute $refs to be relative. They need to be absolute for Monaco to resolve them
      // when the schema is in a oneOf (to be merged with extension schemas).
      schema = schema.replace(
        /https:\/\/sourcegraph\.com\/v1\/settings\.schema\.json#\/definitions\//g,
        '#/definitions/'
      )

      const types = await compileJSONSchema(JSON.parse(schema), 'settings.schema', {
        cwd: schemaDirectory,
        $refOptions: {
          resolve: /** @type {import('json-schema-ref-parser').Options['resolve']} */ ({
            draftV7resolver,
            // there should be no reason to make network calls during this process,
            // and if there are we've broken env for offline devs/increased dev startup time
            http: false,
          }),
        },
      })
      await writeFile(path.join(outputDirectory, `${file}.schema.d.ts`), types)
    })
  )
}

function watchSchema() {
  return gulp.watch(path.join(__dirname, '../schema/*.schema.json'), schema)
}

module.exports = {
  watchSchema,
  schema,
  graphQlSchema,
  watchGraphQlSchema,
  graphQlOperations,
  watchGraphQlOperations,
}
