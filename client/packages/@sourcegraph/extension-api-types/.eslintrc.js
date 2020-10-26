const baseConfig = require('../../../../.eslintrc')
module.exports = {
  extends: '../../../../.eslintrc.js',
  parserOptions: {
    ...baseConfig.parserOptions,
    project: __dirname + '/tsconfig.json',
  },
  overrides: baseConfig.overrides,
}
