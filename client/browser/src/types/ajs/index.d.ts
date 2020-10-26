interface AJS {
    /**
     * The AJS.contextPath() function returns the "path" to the application,
     * which is needed when creating absolute urls within the application.
     */
    contextPath(): string
}

interface Window {
    /**
     * The `AJS` global object provides helper methods to interact with
     * the UI of Atlassian products. It is only defined when executing in the
     * Bitbucket Server native integration.
     */
    AJS?: AJS
}
