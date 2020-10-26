import configure from 'core-js/configurator'

configure({
    usePolyfill: [
        // Polyfill URL because Chrome and Firefox are not spec-compliant
        // Hostnames of URIs with custom schemes (e.g. git) are not parsed out
        'URL',
        // URLSearchParams.prototype.keys() is not iterable in Firefox
        'URLSearchParams',
    ],
})
