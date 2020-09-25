module.exports = {
  plugins: [
    require('postcss-preset-env')(),
    // not included in preset-env: https://github.com/cssnano/cssnano/issues/945
    require('postcss-normalize-display-values'),
  ],
}
