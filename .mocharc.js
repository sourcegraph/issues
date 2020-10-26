module.exports = {
  require: ['ts-node/register', 'abort-controller/polyfill', __dirname + '/client/shared/dev/fetch'],
  extension: ['js', 'ts'],
  // 1 minute test timeout. This must be greater than the default Puppeteer
  // command timeout of 30s in order to get the stack trace to point to the
  // Puppeteer command that failed instead of a cryptic test timeout
  // location.
  timeout: '60s',
  slow: '2s',
}
