// NOTE: Can't use @types/jasmine because it has global type definitions that conflict with jest.

declare var jasmine: {
    getEnv(): {
        addReporter(reporter: { specDone?(result: CustomReporterResult): void })
    }
}

interface CustomReporterResult {
    status: 'passed' | 'failed' | 'disabled' | 'pending'
    fullName: string
}
