// This is a basic setup to do codemods on our TypeScript code.
// For example, this can be used to make a large refactor that requires passing a new prop through a lot of components.
// We can go through all type errors TypeScript reports and add the missing prop to JSX and interfaces.
// The script may need to be run multiple times to fix new type errors, and some cases may need to be updated manually.
// Overall, this can be much faster for those large refactors than making every change by hand.
// This script can be run in the VS Code debugger by selecting "TS-Morph".
// Learn more about how to use ts-morph in the documentation: https://ts-morph.com/manipulation/
// To understand the AST for a given file, paste it into https://ts-ast-viewer.com/
// The code committed here is just as an example that can be modified locally.
// Code mods don't have to be committed to the repo (unless they could be useful as a reference too).

import { Project, QuoteKind } from 'ts-morph'
import { formatSourceFile } from './prettier-ts-morph'
import { addMissingHistoryProp } from './add-history-prop'
import * as path from 'path'

async function main(): Promise<void> {
    const repoRoot = path.resolve(__dirname, '..', '..')

    const project = new Project({
        tsConfigFilePath: path.resolve(repoRoot, 'tsconfig.json'),
        manipulationSettings: {
            quoteKind: QuoteKind.Single,
            useTrailingCommas: true,
        },
    })
    // project.enableLogging(true)
    project.addSourceFilesFromTsConfig(path.resolve(repoRoot, 'web/tsconfig.json'))
    project.addSourceFilesFromTsConfig(path.resolve(repoRoot, 'shared/tsconfig.json'))
    project.addSourceFilesAtPaths([
        path.resolve(repoRoot, 'client/web/src/**/*.d.ts'),
        path.resolve(repoRoot, 'client/shared/src/**/*.d.ts'),
        path.resolve(repoRoot, 'client/browser/src/**/*.d.ts'),
    ])

    console.log('Getting diagnostics')
    const diagnostics = project
        .getPreEmitDiagnostics()
        // Some declaration files are not found by ts-morph for some reason, ignore
        .filter(d => !/(declaration|type definition) file/i.test(project.formatDiagnosticsWithColorAndContext([d])))

    for (const diagnostic of diagnostics.filter(d =>
        /property 'history' is missing/i.test(project.formatDiagnosticsWithColorAndContext([d]))
    )) {
        try {
            let sourceFile = diagnostic.getSourceFile()
            if (!sourceFile) {
                continue
            }
            addMissingHistoryProp(diagnostic, sourceFile)
            sourceFile = await formatSourceFile(sourceFile)
            await sourceFile.save()
        } catch (error) {
            console.error(error)
        }
    }
}

// eslint-disable-next-line @typescript-eslint/no-floating-promises
main()
