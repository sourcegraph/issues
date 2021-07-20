package assets

type WebpackManifest struct {
	// AppJSBundlePath contains the file name of the main
	// Webpack bundle that serves as the entrypoint
	// for the webapp code.
	AppJSBundlePath string `json:"app.js"`
	// AppJSRuntimeBundlePath contains the file name of the
	// JS runtime bundle which is served only in development environment
	// to kick off the main application code.
	AppJSRuntimeBundlePath *string `json:"runtime.js"`
	// Main CSS bundle, only present in production.
	AppCSSBundlePath *string `json:"app.css"`
}
