import * as Monaco from 'monaco-editor'

import { SearchPatternType } from '../../graphql-operations'

import { toMonacoRange } from './decoratedToken'
import { validateFilter } from './filters'
import { Token } from './token'

/**
 * Returns the diagnostics for a scanned search query to be displayed in the Monaco query input.
 */
export function getDiagnostics(tokens: Token[], patternType: SearchPatternType): Monaco.editor.IMarkerData[] {
    const diagnostics: Monaco.editor.IMarkerData[] = []
    for (const token of tokens) {
        if (token.type === 'filter') {
            const { field, value } = token
            const validationResult = validateFilter(field.value, value)
            if (validationResult.valid) {
                continue
            }
            diagnostics.push({
                severity: Monaco.MarkerSeverity.Error,
                message: validationResult.reason,
                ...toMonacoRange(field.range),
            })
        }
    }
    return diagnostics
}
