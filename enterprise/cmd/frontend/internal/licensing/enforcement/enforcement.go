package enforcement

import (
	"fmt"
	"html"
	"net/http"
)

// WriteSubscriptionErrorResponseForFeature is a wrapper around WriteSubscriptionErrorResponse that
// generates the error title and message indicating that the current license does not active the
// given feature.
func WriteSubscriptionErrorResponseForFeature(w http.ResponseWriter, featureNameHumanReadable string) {
	WriteSubscriptionErrorResponse(
		w, http.StatusForbidden,
		fmt.Sprintf("License is not valid for %s", featureNameHumanReadable),
		fmt.Sprintf("To use the %s feature, a site admin must upgrade the Sourcegraph license in the Sourcegraph [**site configuration**](/site-admin/configuration). (The site admin may also remove the site configuration that enables this feature to dismiss this message.)", featureNameHumanReadable),
	)
}

// WriteSubscriptionErrorResponse writes an HTTP response that displays a standalone error page to
// the user.
//
// The title and message should be full sentences that describe the problem and how to fix it. Use
// WriteSubscriptionErrorResponseForFeature to generate these for the common case of a failed
// license feature check.
func WriteSubscriptionErrorResponse(w http.ResponseWriter, statusCode int, title, message string) {
	w.WriteHeader(statusCode)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Inline all styles and resources because those requests will fail (our middleware
	// intercepts all HTTP requests).
	fmt.Fprintln(w, `
<title>`+html.EscapeString(title)+` - Sourcegraph</title>
<style>
.bg {
	position: absolute;
	user-select: none;
	pointer-events: none;
	z-index: -1;
	top: 0;
	bottom: 0;
	left: 0;
	right: 0;
	/* The Sourcegraph logo in SVG. */
	background-image: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 124 127"><g fill="none" fill-rule="evenodd"><path fill="%23F96316" d="M35.942 16.276L63.528 117.12c1.854 6.777 8.85 10.768 15.623 8.912 6.778-1.856 10.765-8.854 8.91-15.63L60.47 9.555C58.615 2.78 51.62-1.212 44.847.645c-6.772 1.853-10.76 8.853-8.905 15.63z"/><path fill="%23B200F8" d="M87.024 15.894L17.944 93.9c-4.66 5.26-4.173 13.303 1.082 17.964 5.255 4.66 13.29 4.174 17.95-1.084l69.08-78.005c4.66-5.26 4.173-13.3-1.082-17.962-5.257-4.664-13.294-4.177-17.95 1.08z"/><path fill="%2300B4F2" d="M8.75 59.12l98.516 32.595c6.667 2.205 13.86-1.414 16.065-8.087 2.21-6.672-1.41-13.868-8.08-16.076L16.738 34.96c-6.67-2.207-13.86 1.412-16.066 8.085-2.204 6.672 1.416 13.87 8.08 16.075z"/></g></svg>');
	background-repeat: repeat;
	background-size: 5rem;
	opacity: 0.1;
}

.msg {
	font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
	max-width: 30rem;
	margin: 20vh auto 0;
	border: solid 2px rgba(255,0,0,0.8);
	background-color: rgba(255,255,255,0.8);
	color: rgb(30, 0, 0);
	padding: 1rem 2rem;
}

h1 {
	font-size: 1.5rem;
}
</style>
<meta name="robots" content="noindex">
<body>
<div class=bg></div>
<div class=msg><h1>`+html.EscapeString(title)+`</h1><p>`+html.EscapeString(message)+`</p><p>See <a href="https://about.sourcegraph.com/pricing">about.sourcegraph.com</a> for more information.</p></div>`)
}
