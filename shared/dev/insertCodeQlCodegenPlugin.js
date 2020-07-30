// @ts-check

/**
 *
 * @param {import('graphql').GraphQLSchema} schema
 * @param {import('@graphql-codegen/plugin-helpers').Types.DocumentFile[]} documents
 * @param {{insertCodeSnippet?: string}} config
 */
const plugin = (schema, documents, config) => {
  const { insertCodeSnippet = '' } = config
  return insertCodeSnippet
}

module.exports = { plugin }
