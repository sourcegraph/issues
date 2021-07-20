module.exports = {
  schemaPaths: ['cmd/frontend/graphqlbackend/*.graphql'],
  rules: [
    'deprecations-have-a-reason',
    'fields-have-descriptions',
    'input-object-values-have-descriptions',
    'types-have-descriptions',
    'relay-connection-types-spec',
    'relay-connection-arguments-spec',
  ],
}
