// @ts-check

const config = require('../../jest.config.base')

/** @type {jest.InitialOptions} */
const exportedConfig = {
  ...config,
  displayName: 'web',
  rootDir: __dirname,
}

module.exports = exportedConfig
