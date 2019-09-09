import * as fs from 'mz/fs'
import * as temp from 'temp'
import * as zlib from 'mz/zlib'
import { ConnectionCache, DocumentCache, ResultChunkCache } from './cache'
import { createBackend } from './backend'
import { lsp } from 'lsif-protocol'
import { Readable } from 'stream'

describe('Database', () => {
    let storageRoot!: string
    const connectionCache = new ConnectionCache(10)
    const documentCache = new DocumentCache(10)
    const resultChunkCache = new ResultChunkCache(10)

    beforeAll(async () => {
        storageRoot = temp.mkdirSync('typescript') // eslint-disable-line no-sync
        const backend = await createBackend(storageRoot, connectionCache, documentCache, resultChunkCache)
        const inputs: { input: Readable; repository: string; commit: string }[] = []

        for (const repository of ['a', 'b1', 'b2', 'b3', 'c1', 'c2', 'c3']) {
            const input = fs
                .createReadStream(`./test-data/typescript/data/${repository}.lsif.gz`)
                .pipe(zlib.createGunzip())
            const commit = createCommit(repository)
            inputs.push({ input, repository, commit })
        }

        for (const { input, repository, commit } of inputs) {
            await backend.insertDump(input, repository, commit)
        }
    })

    it('should find all defs of `add` from repo a', async () => {
        const backend = await createBackend(storageRoot, connectionCache, documentCache, resultChunkCache)
        const db = await backend.createDatabase('a', createCommit('a'))
        const definitions = await db.definitions('src/index.ts', { line: 11, character: 18 })
        expect(definitions).toContainEqual(createLocation('src/index.ts', 0, 16, 0, 19))
        expect(definitions && definitions.length).toEqual(1)
    })

    it('should find all defs of `add` from repo b1', async () => {
        const backend = await createBackend(storageRoot, connectionCache, documentCache, resultChunkCache)
        const db = await backend.createDatabase('b1', createCommit('b1'))
        const definitions = await db.definitions('src/index.ts', { line: 3, character: 12 })
        expect(definitions).toContainEqual(createRemoteLocation('a', 'src/index.ts', 0, 16, 0, 19))
        expect(definitions && definitions.length).toEqual(1)
    })

    it('should find all defs of `mul` from repo b1', async () => {
        const backend = await createBackend(storageRoot, connectionCache, documentCache, resultChunkCache)
        const db = await backend.createDatabase('b1', createCommit('b1'))
        const definitions = await db.definitions('src/index.ts', { line: 3, character: 16 })
        expect(definitions).toContainEqual(createRemoteLocation('a', 'src/index.ts', 4, 16, 4, 19))
        expect(definitions && definitions.length).toEqual(1)
    })

    it('should find all refs of `mul` from repo a', async () => {
        const backend = await createBackend(storageRoot, connectionCache, documentCache, resultChunkCache)
        const db = await backend.createDatabase('a', createCommit('a'))
        // TODO - (FIXME) why are these garbage results in the index
        const references = (await db.references('src/index.ts', { line: 4, character: 19 }))!.filter(
            l => !l.uri.includes('node_modules')
        )

        // TODO - should the definition be in this result set?
        expect(references).toContainEqual(createLocation('src/index.ts', 4, 16, 4, 19)) // def
        expect(references).toContainEqual(createRemoteLocation('b1', 'src/index.ts', 0, 14, 0, 17)) // import
        expect(references).toContainEqual(createRemoteLocation('b1', 'src/index.ts', 3, 15, 3, 18)) // 1st use
        expect(references).toContainEqual(createRemoteLocation('b1', 'src/index.ts', 3, 26, 3, 29)) // 2nd use
        expect(references).toContainEqual(createRemoteLocation('b2', 'src/index.ts', 0, 14, 0, 17)) // import
        expect(references).toContainEqual(createRemoteLocation('b2', 'src/index.ts', 3, 15, 3, 18)) // 1st use
        expect(references).toContainEqual(createRemoteLocation('b2', 'src/index.ts', 3, 26, 3, 29)) // 2nd use
        expect(references).toContainEqual(createRemoteLocation('b3', 'src/index.ts', 0, 14, 0, 17)) // import
        expect(references).toContainEqual(createRemoteLocation('b3', 'src/index.ts', 3, 15, 3, 18)) // 1st use
        expect(references).toContainEqual(createRemoteLocation('b3', 'src/index.ts', 3, 26, 3, 29)) // 2nd use

        // Ensure no additional references
        expect(references && references.length).toEqual(10)
    })

    it('should find all refs of `mul` from repo b1', async () => {
        const backend = await createBackend(storageRoot, connectionCache, documentCache, resultChunkCache)
        const db = await backend.createDatabase('b1', createCommit('b1'))
        // TODO - (FIXME) why are these garbage results in the index
        const references = (await db.references('src/index.ts', { line: 3, character: 16 }))!.filter(
            l => !l.uri.includes('node_modules')
        )

        // TODO - should the definition be in this result set?
        expect(references).toContainEqual(createRemoteLocation('a', 'src/index.ts', 4, 16, 4, 19)) // def
        expect(references).toContainEqual(createLocation('src/index.ts', 0, 14, 0, 17)) // import
        expect(references).toContainEqual(createLocation('src/index.ts', 3, 15, 3, 18)) // 1st use
        expect(references).toContainEqual(createLocation('src/index.ts', 3, 26, 3, 29)) // 2nd use
        expect(references).toContainEqual(createRemoteLocation('b2', 'src/index.ts', 0, 14, 0, 17)) // import
        expect(references).toContainEqual(createRemoteLocation('b2', 'src/index.ts', 3, 15, 3, 18)) // 1st use
        expect(references).toContainEqual(createRemoteLocation('b2', 'src/index.ts', 3, 26, 3, 29)) // 2nd use
        expect(references).toContainEqual(createRemoteLocation('b3', 'src/index.ts', 0, 14, 0, 17)) // import
        expect(references).toContainEqual(createRemoteLocation('b3', 'src/index.ts', 3, 15, 3, 18)) // 1st use
        expect(references).toContainEqual(createRemoteLocation('b3', 'src/index.ts', 3, 26, 3, 29)) // 2nd use

        // Ensure no additional references
        expect(references && references.length).toEqual(10)
    })

    it('should find all refs of `add` from repo a', async () => {
        const backend = await createBackend(storageRoot, connectionCache, documentCache, resultChunkCache)
        const db = await backend.createDatabase('a', createCommit('a'))
        // TODO - (FIXME) why are these garbage results in the index
        const references = (await db.references('src/index.ts', { line: 0, character: 17 }))!.filter(
            l => !l.uri.includes('node_modules')
        )

        // TODO - should the definition be in this result set?
        expect(references).toContainEqual(createLocation('src/index.ts', 0, 16, 0, 19)) // def
        expect(references).toContainEqual(createLocation('src/index.ts', 11, 18, 11, 21)) // 1st use
        expect(references).toContainEqual(createRemoteLocation('b1', 'src/index.ts', 0, 9, 0, 12)) // import
        expect(references).toContainEqual(createRemoteLocation('b1', 'src/index.ts', 3, 11, 3, 14)) // 1st use
        expect(references).toContainEqual(createRemoteLocation('b2', 'src/index.ts', 0, 9, 0, 12)) // import
        expect(references).toContainEqual(createRemoteLocation('b2', 'src/index.ts', 3, 11, 3, 14)) // 1st use
        expect(references).toContainEqual(createRemoteLocation('b3', 'src/index.ts', 0, 9, 0, 12)) // import
        expect(references).toContainEqual(createRemoteLocation('b3', 'src/index.ts', 3, 11, 3, 14)) // 1st use
        expect(references).toContainEqual(createRemoteLocation('c1', 'src/index.ts', 0, 9, 0, 12)) // import
        expect(references).toContainEqual(createRemoteLocation('c1', 'src/index.ts', 3, 11, 3, 14)) // 1st use
        expect(references).toContainEqual(createRemoteLocation('c1', 'src/index.ts', 3, 15, 3, 18)) // 2nd use
        expect(references).toContainEqual(createRemoteLocation('c1', 'src/index.ts', 3, 26, 3, 29)) // 3rd use
        expect(references).toContainEqual(createRemoteLocation('c2', 'src/index.ts', 0, 9, 0, 12)) // import
        expect(references).toContainEqual(createRemoteLocation('c2', 'src/index.ts', 3, 11, 3, 14)) // 1st use
        expect(references).toContainEqual(createRemoteLocation('c2', 'src/index.ts', 3, 15, 3, 18)) // 2nd use
        expect(references).toContainEqual(createRemoteLocation('c2', 'src/index.ts', 3, 26, 3, 29)) // 3rd use
        expect(references).toContainEqual(createRemoteLocation('c3', 'src/index.ts', 0, 9, 0, 12)) // import
        expect(references).toContainEqual(createRemoteLocation('c3', 'src/index.ts', 3, 11, 3, 14)) // 1st use
        expect(references).toContainEqual(createRemoteLocation('c3', 'src/index.ts', 3, 15, 3, 18)) // 2nd use
        expect(references).toContainEqual(createRemoteLocation('c3', 'src/index.ts', 3, 26, 3, 29)) // 3rd use

        // Ensure no additional references
        expect(references && references.length).toEqual(20)
    })

    it('should find all refs of `add` from repo c1', async () => {
        const backend = await createBackend(storageRoot, connectionCache, documentCache, resultChunkCache)
        const db = await backend.createDatabase('c1', createCommit('c1'))
        // TODO - (FIXME) why are these garbage results in the index
        const references = (await db.references('src/index.ts', { line: 3, character: 16 }))!.filter(
            l => !l.uri.includes('node_modules')
        )

        // TODO - should the definition be in this result set?
        expect(references).toContainEqual(createRemoteLocation('a', 'src/index.ts', 0, 16, 0, 19)) // def
        expect(references).toContainEqual(createRemoteLocation('a', 'src/index.ts', 11, 18, 11, 21)) // 1st use
        expect(references).toContainEqual(createRemoteLocation('b1', 'src/index.ts', 0, 9, 0, 12)) // import
        expect(references).toContainEqual(createRemoteLocation('b1', 'src/index.ts', 3, 11, 3, 14)) // 1st use
        expect(references).toContainEqual(createRemoteLocation('b2', 'src/index.ts', 0, 9, 0, 12)) // import
        expect(references).toContainEqual(createRemoteLocation('b2', 'src/index.ts', 3, 11, 3, 14)) // 1st use
        expect(references).toContainEqual(createRemoteLocation('b3', 'src/index.ts', 0, 9, 0, 12)) // import
        expect(references).toContainEqual(createRemoteLocation('b3', 'src/index.ts', 3, 11, 3, 14)) // 1st use
        expect(references).toContainEqual(createLocation('src/index.ts', 0, 9, 0, 12)) // import
        expect(references).toContainEqual(createLocation('src/index.ts', 3, 11, 3, 14)) // 1st use
        expect(references).toContainEqual(createLocation('src/index.ts', 3, 15, 3, 18)) // 2nd use
        expect(references).toContainEqual(createLocation('src/index.ts', 3, 26, 3, 29)) // 3rd use
        expect(references).toContainEqual(createRemoteLocation('c2', 'src/index.ts', 0, 9, 0, 12)) // import
        expect(references).toContainEqual(createRemoteLocation('c2', 'src/index.ts', 3, 11, 3, 14)) // 1st use
        expect(references).toContainEqual(createRemoteLocation('c2', 'src/index.ts', 3, 15, 3, 18)) // 2nd use
        expect(references).toContainEqual(createRemoteLocation('c2', 'src/index.ts', 3, 26, 3, 29)) // 3rd use
        expect(references).toContainEqual(createRemoteLocation('c3', 'src/index.ts', 0, 9, 0, 12)) // import
        expect(references).toContainEqual(createRemoteLocation('c3', 'src/index.ts', 3, 11, 3, 14)) // 1st use
        expect(references).toContainEqual(createRemoteLocation('c3', 'src/index.ts', 3, 15, 3, 18)) // 2nd use
        expect(references).toContainEqual(createRemoteLocation('c3', 'src/index.ts', 3, 26, 3, 29)) // 3rd use

        // Ensure no additional references
        expect(references && references.length).toEqual(20)
    })
})

//
// Helpers

function createLocation(
    uri: string,
    startLine: number,
    startCharacter: number,
    endLine: number,
    endCharacter: number
): lsp.Location {
    return lsp.Location.create(uri, {
        start: { line: startLine, character: startCharacter },
        end: { line: endLine, character: endCharacter },
    })
}

function createRemoteLocation(
    repository: string,
    path: string,
    startLine: number,
    startCharacter: number,
    endLine: number,
    endCharacter: number
): lsp.Location {
    return createLocation(
        `git://${repository}?${createCommit(repository)}#${path}`,
        startLine,
        startCharacter,
        endLine,
        endCharacter
    )
}

function createCommit(repository: string): string {
    return repository.repeat(40).substring(0, 40)
}
