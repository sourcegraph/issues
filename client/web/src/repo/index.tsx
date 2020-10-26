/**
 * Performs a redirect to the host of the given URL with the path, query etc. properties of the current URL.
 */
export function redirectToExternalHost(externalRedirectURL: string): void {
    const externalHostURL = new URL(externalRedirectURL)
    const redirectURL = new URL(window.location.href)
    // Preserve the path of the current URL and redirect to the repo on the external host.
    redirectURL.host = externalHostURL.host
    redirectURL.protocol = externalHostURL.protocol
    window.location.replace(redirectURL.href)
}
