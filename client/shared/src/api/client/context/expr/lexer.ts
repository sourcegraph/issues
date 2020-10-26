/** Valid token types in expressions. */
export enum TokenType {
    /** An operator. */
    Operator,

    /** An identifier. */
    Identifier,

    /** A string literal. */
    String,

    /**
     * The start of a template until its first expression.
     *
     * See https://tc39.github.io/ecma262/#sec-template-literal-lexical-components for documentation on the
     * ECMAScript lexical components for templates, upon which this is based.
     */
    TemplateHead,

    /** The end of a previous template expression until the next template expression. */
    TemplateMiddle,

    /** The end of a previous template expression until the end of the template. */
    TemplateTail,

    /** A template with no substitutions. */
    NoSubstitutionTemplate,

    /** A number literal. */
    Number,
}

/** A token that the expression lexer scanned in an expression. */
export interface Token {
    /** The type of this token. */
    type: TokenType

    /**
     * The token's value.
     *
     * For string and template literals, this is the parsed string value (after accounting for escape sequences but
     * not template expressions). For number literals, this is the (unparsed) string representation.
     */
    value: any

    /** The start character position of this token. */
    start: number

    /** The end character position of this token. */
    end: number
}

/**
 * All valid operators in expressions. The values are the operator precedence (or, for operators that are not operators, 0). This
 * must be kept in sync with OPERATOR_CHARS.
 *
 * Exported for testing only.
 */
export const OPERATORS = {
    '(': 0,
    ')': 0,
    '}': 0,
    ',': 0,
    '=': 0,
    '||': 1,
    '&&': 2,
    '^': 4,
    '==': 6,
    '!=': 6,
    '===': 6,
    '!==': 6,
    '<': 7,
    '>': 7,
    '<=': 7,
    '>=': 7,
    '+': 9,
    '-': 9,
    '*': 10,
    '/': 10,
    '%': 10,
    '!': 11,
}

/** All valid operators. */
export type Operator = keyof typeof OPERATORS

export type OperatorTree = boolean | { [ch: string]: OperatorTree }

/**
 * A tree with the next valid operator characters for multi-character operators. This must be kept in sync with
 * OPERATORS.
 *
 * Exported for testing only.
 */
export const OPERATOR_CHARS: { [ch: string]: OperatorTree } = {
    '&': { '&': true },
    '|': { '|': true },
    '=': {
        '\u0000': true,
        '=': {
            '\u0000': true,
            '=': true,
        },
    },
    '!': {
        '\u0000': true,
        '=': {
            '\u0000': true,
            '=': true,
        },
    },
    '<': { '\u0000': true, '=': true },
    '>': { '\u0000': true, '=': true },
    '^': true,
    '}': true,
    '(': true,
    ')': true,
    ',': true,
    '+': true,
    '-': true,
    '*': true,
    '/': true,
    '%': true,
}

function isWhiteSpace(character: string): boolean {
    return character === '\u0009' || character === ' ' || character === '\u00A0'
}

function isLetter(character: string): boolean {
    return (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z')
}

function isDecimalDigit(character: string): boolean {
    return character >= '0' && character <= '9'
}

function isIdentifierStart(character: string): boolean {
    return character === '_' || isLetter(character)
}

function isIdentifierPart(character: string): boolean {
    return isIdentifierStart(character) || isDecimalDigit(character) || character === '.'
}

/** Scans an expression. */
export class Lexer {
    private expression = ''
    private length = 0
    protected _index = 0
    private marker = 0
    protected curlyStack = 0

    /** The current character position of the lexer's cursor. */
    public get index(): number {
        return this._index
    }

    public reset(string: string): void {
        this.expression = string
        this.length = string.length
        this._index = 0
        this.curlyStack = 0
    }

    public next(): Token | undefined {
        this.skipSpaces()
        if (this._index >= this.length) {
            return undefined
        }

        this.marker = this._index

        const token = this.scanNext()
        if (token !== undefined) {
            return token
        }

        throw new SyntaxError(`Unexpected character ${JSON.stringify(this.peekNextChar())} (at ${this.index})`)
    }

    public peek(): Omit<Token, 'start' | 'end'> | undefined {
        const savedIndex = this._index
        const savedCurlyStack = this.curlyStack
        let token: Token | undefined
        try {
            token = this.next()
        } catch {
            token = undefined
        }
        this._index = savedIndex
        this.curlyStack = savedCurlyStack

        if (!token) {
            return undefined
        }
        return { type: token.type, value: token.value }
    }

    protected scanNext(): Token | undefined {
        let token = this.scanString()
        if (token !== undefined) {
            return token
        }

        token = this.scanTemplate()
        if (token !== undefined) {
            return token
        }

        token = this.scanNumber()
        if (token !== undefined) {
            return token
        }

        token = this.scanOperator()
        if (token !== undefined) {
            return token
        }

        token = this.scanIdentifier()
        if (token !== undefined) {
            return token
        }

        return undefined
    }

    private peekNextChar(advance = 0): string {
        const index = this._index + advance
        return index < this.length ? this.expression.charAt(index) : '\u0000'
    }

    private getNextChar(): string {
        let character = '\u0000'
        const index = this._index
        if (index < this.length) {
            character = this.expression.charAt(index)
            this._index += 1
        }
        return character
    }

    private createToken(type: TokenType, value: any): Token {
        return {
            type,
            value,
            start: this.marker,
            end: this._index,
        }
    }

    private skipSpaces(): void {
        while (this._index < this.length) {
            const character = this.peekNextChar()
            if (!isWhiteSpace(character)) {
                break
            }
            this.getNextChar()
        }
    }

    private scanOperator(): Token | undefined {
        let searchTree: OperatorTree | boolean = OPERATOR_CHARS
        let value = ''
        while (searchTree && searchTree !== true) {
            const character = this.peekNextChar()
            searchTree = searchTree[character]
            if (searchTree) {
                value += character
                this.getNextChar()
            }
        }
        if (value === '}') {
            this.curlyStack--
        }
        if (value === '') {
            return undefined
        }
        return this.createToken(TokenType.Operator, value)
    }

    private scanIdentifier(): Token | undefined {
        let character = this.peekNextChar()
        if (!isIdentifierStart(character)) {
            return undefined
        }

        let id = this.getNextChar()
        while (true) {
            character = this.peekNextChar()
            if (!isIdentifierPart(character)) {
                break
            }
            id += this.getNextChar()
        }

        return this.createToken(TokenType.Identifier, id)
    }

    private scanString(): Token | undefined {
        const quote = this.peekNextChar()
        if (quote !== "'" && quote !== '"') {
            return undefined
        }
        this.getNextChar()

        let terminated = false
        let string = ''
        while (this._index < this.length) {
            const character = this.getNextChar()
            if (character === quote) {
                terminated = true
                break
            }
            if (character === '\\') {
                string += backslashEscapeCodeString(this.getNextChar())
            } else {
                string += character
            }
        }
        if (!terminated) {
            throw new Error(`Unterminated string literal (at ${this.index})`)
        }
        return this.createToken(TokenType.String, string)
    }

    private scanTemplate(): Token | undefined {
        const character = this.peekNextChar()
        if (!(character === '`' || (character === '}' && this.curlyStack > 0))) {
            return undefined
        }
        this.getNextChar()

        const head = character === '`'
        return this.doScanTemplate(head)
    }

    protected backtick(): boolean {
        return true
    }

    protected doScanTemplate(head: boolean): Token {
        let tail = false

        let terminated = false
        let hasSubstitution = false
        let string = ''
        while (this._index < this.length) {
            const character = this.getNextChar()
            if (character === '`' && this.backtick()) {
                tail = true
                terminated = true
                break
            }
            if (character === '\\') {
                string += backslashEscapeCodeString(this.getNextChar())
            } else {
                if (character === '$') {
                    const character2 = this.peekNextChar()
                    if (character2 === '{') {
                        this.curlyStack++
                        this.getNextChar()
                        terminated = true
                        hasSubstitution = true
                        break
                    }
                }
                string += character
            }
        }
        if (!head) {
            this.curlyStack--
        }
        if (this.backtick()) {
            if (!terminated) {
                throw new Error(`Unterminated template literal (at ${this.index})`)
            }
        } else if (this._index === this.length) {
            tail = true
        }

        let type: TokenType
        if (head && terminated && !hasSubstitution) {
            type = TokenType.NoSubstitutionTemplate
        } else if (head) {
            type = TokenType.TemplateHead
        } else if (tail) {
            type = TokenType.TemplateTail
        } else {
            type = TokenType.TemplateMiddle
        }
        return this.createToken(type, string)
    }

    private scanNumber(): Token | undefined {
        let character = this.peekNextChar()
        if (!isDecimalDigit(character) && character !== '.') {
            return undefined
        }

        let number = ''
        if (character !== '.') {
            number = this.getNextChar()
            while (true) {
                character = this.peekNextChar()
                if (!isDecimalDigit(character)) {
                    break
                }
                number += this.getNextChar()
            }
        }

        if (character === '.') {
            number += this.getNextChar()
            while (true) {
                character = this.peekNextChar()
                if (!isDecimalDigit(character)) {
                    break
                }
                number += this.getNextChar()
            }
        }

        if (character === 'e' || character === 'E') {
            number += this.getNextChar()
            character = this.peekNextChar()
            if (character === '+' || character === '-' || isDecimalDigit(character)) {
                number += this.getNextChar()
                while (true) {
                    character = this.peekNextChar()
                    if (!isDecimalDigit(character)) {
                        break
                    }
                    number += this.getNextChar()
                }
            } else {
                character = `character ${JSON.stringify(character)}`
                if (this._index >= this.length) {
                    character = '<end>'
                }
                throw new SyntaxError(`Unexpected ${character} after the exponent sign (at ${this.index})`)
            }
        }

        if (number === '.') {
            throw new SyntaxError(`Expected decimal digits after the dot sign (at ${this.index})`)
        }

        return this.createToken(TokenType.Number, number)
    }
}

/** Scans a template. */
export class TemplateLexer extends Lexer {
    public next(): Token | undefined {
        if (this._index === 0) {
            return this.doScanTemplate(true)
        }
        return super.next()
    }

    protected backtick(): boolean {
        // The root is not surrounded with backticks.
        return this.curlyStack !== 0
    }
}

function backslashEscapeCodeString(character: string): string {
    switch (character) {
        case 'n':
            return '\n'
        case 'r':
            return '\r'
        case 't':
            return '\t'
        default:
            return character
    }
}
