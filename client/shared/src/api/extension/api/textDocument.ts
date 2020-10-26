import { Position, Range } from '@sourcegraph/extension-api-classes'
import * as sourcegraph from 'sourcegraph'
import { PrefixSumComputer } from '../../../util/prefixSumComputer'
import { getWordAtText } from '../../../util/wordHelpers'

/** @internal */
export class ExtensionDocument implements sourcegraph.TextDocument {
    private _eol: string
    private _lines: string[]
    public uri: string
    public languageId: string
    public text: string | undefined

    constructor(private model: Pick<sourcegraph.TextDocument, 'uri' | 'languageId' | 'text'>) {
        this._eol = getEOL(model.text || '')
        this._lines = model.text !== undefined ? model.text.split(this._eol) : []
        this.uri = this.model.uri
        this.languageId = this.model.languageId
        this.text = this.model.text
    }

    public update({ text }: Pick<sourcegraph.TextDocument, 'text'>): void {
        this.model = {
            ...this.model,
            text,
        }
        this._eol = getEOL(text || '')
        this._lines = text !== undefined ? text.split(this._eol) : []
        this.text = text
    }

    public offsetAt(position: sourcegraph.Position): number {
        this.throwIfNoModelText()
        position = this.validatePosition(position)
        return this.lineStarts.getAccumulatedValue(position.line - 1) + position.character
    }

    public positionAt(offset: number): sourcegraph.Position {
        this.throwIfNoModelText()
        offset = Math.floor(offset)
        offset = Math.max(0, offset)

        const out = this.lineStarts.getIndexOf(offset)
        const lineLength = this._lines[out.index].length
        const character = Math.min(out.remainder, lineLength) // ensure we return a valid position
        return new Position(out.index, character)
    }

    public validatePosition(position: sourcegraph.Position): sourcegraph.Position {
        this.throwIfNoModelText()
        if (!(position instanceof Position)) {
            throw new TypeError('invalid argument')
        }

        let { line, character } = position
        let hasChanged = false

        if (line < 0) {
            line = 0
            character = 0
            hasChanged = true
        } else if (line >= this._lines.length) {
            line = this._lines.length - 1
            character = this._lines[line].length
            hasChanged = true
        } else {
            const maxCharacter = this._lines[line].length
            if (character < 0) {
                character = 0
                hasChanged = true
            } else if (character > maxCharacter) {
                character = maxCharacter
                hasChanged = true
            }
        }

        if (!hasChanged) {
            return position
        }
        return new Position(line, character)
    }

    public validateRange(range: sourcegraph.Range): sourcegraph.Range {
        this.throwIfNoModelText()
        if (!(range instanceof Range)) {
            throw new TypeError('invalid argument')
        }

        const start = this.validatePosition(range.start)
        const end = this.validatePosition(range.end)

        if (start === range.start && end === range.end) {
            return range
        }
        return new Range(start.line, start.character, end.line, end.character)
    }

    public getWordRangeAtPosition(position: sourcegraph.Position): sourcegraph.Range | undefined {
        this.throwIfNoModelText()
        position = this.validatePosition(position)
        const wordAtText = getWordAtText(position.character, this._lines[position.line])
        if (wordAtText) {
            return new Range(position.line, wordAtText.startColumn, position.line, wordAtText.endColumn)
        }
        return undefined
    }

    public getText(range?: sourcegraph.Range): string | undefined {
        this.throwIfNoModelText()
        range = range ? this.validateRange(range) : undefined
        if (!range) {
            return this.text
        }
        const { start, end } = range

        if (start.line === end.line) {
            return this._lines[start.line].slice(start.character, end.character)
        }

        let text = ''
        for (let line = start.line; line <= end.line; line++) {
            if (line === start.line) {
                text += this._lines[line].slice(start.character)
            } else if (line === end.line) {
                text += this._lines[line].slice(0, end.character)
            } else {
                text += this._lines[line]
            }
        }

        return text
    }

    // Memoize computation of line starts.
    private _lineStarts: PrefixSumComputer | null = null
    private get lineStarts(): PrefixSumComputer {
        if (!this._lineStarts) {
            const eolLength = this._eol.length
            const linesLength = this._lines.length
            const lineStartValues = new Uint32Array(linesLength)
            for (let index = 0; index < linesLength; index++) {
                lineStartValues[index] = this._lines[index].length + eolLength
            }
            this._lineStarts = new PrefixSumComputer(lineStartValues)
        }
        return this._lineStarts
    }

    private throwIfNoModelText(): void {
        if (this.model.text === undefined) {
            throw new Error('model text is not available')
        }
    }

    public toJSON(): any {
        return this.model
    }
}

/**
 * Detects the end-of-line character in the text (either \n, \r\n, or \r).
 */
export function getEOL(text: string): string {
    for (let index = 0; index < text.length; index++) {
        const character = text.charAt(index)
        if (character === '\r') {
            if (index + 1 < text.length && text.charAt(index + 1) === '\n') {
                return '\r\n'
            }
            return '\r'
        }
        if (character === '\n') {
            return '\n'
        }
    }
    return '\n'
}
