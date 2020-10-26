// @ts-check

const { visit } = require('graphql')
const path = require('path')
const logger = require('gulplog')

const ROOT_FOLDER = path.resolve(__dirname, '../../')
/**
 *
 * @param {import('graphql').GraphQLSchema} schema
 * @param {import('@graphql-codegen/plugin-helpers').Types.DocumentFile[]} documents
 * @param {{interfaceNameForOperations?: string}} config
 */
const plugin = (schema, documents, config) => {
  const { interfaceNameForOperations = 'AllOperations' } = config

  /** @type {{name: string, location?: string}[]} */
  const allOperations = []

  for (const item of documents) {
    if (item.document) {
      visit(item.document, {
        enter: {
          OperationDefinition: node => {
            if (node.name && node.name.value) {
              allOperations.push({
                name: node.name.value,
                location: item.location && path.relative(ROOT_FOLDER, item.location),
              })
            }
          },
        },
      })
    }
  }

  const interfaceFields = allOperations.map(
    ({ name, location }) =>
      '\n' +
      `/** ${location || 'location not found'} */\n` +
      `${name}: (variables: ${name}Variables) => ${name}Result\n`
  )
  if (interfaceFields.length === 0) {
    logger.warn('No operations found to generate interface ' + interfaceNameForOperations)
  }
  return (
    '\n' + //
    `export interface ${interfaceNameForOperations} {\n` +
    `    ${interfaceFields.join('\n    ')}\n` +
    '}\n'
  )
}

module.exports = { plugin }
