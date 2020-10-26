// @ts-check

// Use same timezone for all test runs. This affects Intl.DateTimeFormat in node.
module.exports = () => {
  process.env.TZ = 'UTC'
}
