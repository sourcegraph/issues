// A shared variable to keep webpack.config and integration tests context configuration in sync.
export const isHotReloadEnabled = process.env.NODE_ENV !== 'production' && process.env.CI !== 'true'
