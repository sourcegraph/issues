interface PageError {
    statusCode: number
    statusText: string
    error: string
    errorID: string
}

interface Window {
    pageError?: PageError
    context: import('./jscontext').SourcegraphContext
    MonacoEnvironment: {
        getWorkerUrl(moduleId: string, label: string): string
    }
}

declare module '*.scss' {
    const cssModule: string
    export default cssModule
}
declare module '*.yaml' {
    const yamlModule: string
    export default yamlModule
}
declare module '*.yml' {
    const ymlModule: string
    export default ymlModule
}
