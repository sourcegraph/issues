import { TextDocumentIdentifier } from '../../../../shared/src/api/client/types/textDocument'
import { TextDocumentPositionParameters } from '../../../../shared/src/api/protocol'
import { AbsoluteRepoFilePosition, FileSpec, RepoSpec, ResolvedRevisionSpec } from '../../../../shared/src/util/url'

export const toTextDocumentIdentifier = (
    position: RepoSpec & ResolvedRevisionSpec & FileSpec
): TextDocumentIdentifier => ({
    uri: `git://${position.repoName}?${position.commitID}#${position.filePath}`,
})

export const toTextDocumentPositionParameters = (
    position: AbsoluteRepoFilePosition
): TextDocumentPositionParameters => ({
    textDocument: toTextDocumentIdentifier(position),
    position: {
        character: position.position.character - 1,
        line: position.position.line - 1,
    },
})
