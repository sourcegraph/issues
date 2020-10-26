import { Lexer, Operator, TemplateLexer, Token, TokenType } from './lexer'

export type ExpressionNode =
    | { FunctionCall: { name: string; args: ExpressionNode[] } }
    | { Identifier: string }
    | { Literal: { type: TokenType.String | TokenType.Number; value: string } }
    | { Template: { parts: ExpressionNode[] } }
    | { Unary: { operator: Operator; expression: ExpressionNode } }
    | { Binary: { operator: Operator; left: ExpressionNode; right: ExpressionNode } }

/**
 * Parses an expression.
 *
 * TODO: Operator precedence is not handled correctly. Use parentheses to be explicit about your desired
 * precedence.
 */
export class Parser {
    protected lexer!: Lexer

    public parse(expressionString: string): ExpressionNode {
        if (!this.lexer) {
            this.lexer = new Lexer()
        }
        this.lexer.reset(expressionString)
        const expression = this.parseExpression()

        const token = this.lexer.next()
        if (token !== undefined) {
            throw new SyntaxError(
                `Unexpected token at end of input: ${JSON.stringify(token.value)} (at ${this.lexer.index})`
            )
        }

        return expression
    }

    // ArgumentList := Expression |
    //                 Expression ',' ArgumentList
    private parseArgumentList(): ExpressionNode[] {
        const args: ExpressionNode[] = []
        while (true) {
            const expression = this.parseExpression()
            if (expression === undefined) {
                throw new Error(
                    `Parse error on token in arguments list: ${JSON.stringify(this.lexer.peek())} (at ${
                        this.lexer.index
                    })`
                )
            }
            args.push(expression)
            const token = this.lexer.peek()
            if (!matchOp(token, ',')) {
                break
            }
            this.lexer.next()
        }
        return args
    }

    // FunctionCall ::= Identifier '(' ')' ||
    //                  Identifier '(' ArgumentList ')'
    private parseFunctionCall(name: string): ExpressionNode {
        let token: Pick<Token, 'type' | 'value'> | undefined = this.lexer.next()
        if (!matchOp(token, '(')) {
            throw new SyntaxError(`Expected "(" in function call ${JSON.stringify(name)} (at ${this.lexer.index})`)
        }

        token = this.lexer.peek()
        const args: ExpressionNode[] = matchOp(token, ')') ? [] : this.parseArgumentList()

        token = this.lexer.next()
        if (!matchOp(token, ')')) {
            throw new SyntaxError(`Expected ")" in function call ${JSON.stringify(name)} (at ${this.lexer.index})`)
        }

        return {
            FunctionCall: {
                name,
                args,
            },
        }
    }

    private parseTemplateParts(): ExpressionNode[] {
        const parts: ExpressionNode[] = []
        while (true) {
            const token: Pick<Token, 'type' | 'value'> | undefined = this.lexer.peek()
            if (!token) {
                break
            }
            if (token.type === TokenType.TemplateTail) {
                if (token.value) {
                    parts.push({ Literal: { type: TokenType.String, value: token.value } })
                }
                this.lexer.next()
                break
            }
            if (matchOp(token, '}')) {
                this.lexer.next()
            } else if (token.type === TokenType.TemplateMiddle) {
                if (token.value) {
                    parts.push({ Literal: { type: TokenType.String, value: token.value } })
                }
                this.lexer.next()
            } else {
                parts.push(this.parseExpression())
            }
        }
        return parts
    }

    protected parseTemplate(): ExpressionNode {
        const token = this.lexer.peek()
        if (token === undefined) {
            throw new SyntaxError(
                `Unexpected termination of expression at beginning of template (at ${this.lexer.index})`
            )
        }

        if (token.type === TokenType.NoSubstitutionTemplate) {
            this.lexer.next()
            // The caller doesn't need to distinguish between NoSubstitutionTemplate and String
            // tokens, so collapse both token types into String.
            return { Literal: { type: TokenType.String, value: token.value } }
        }

        if (token.type === TokenType.TemplateHead || token.type === TokenType.TemplateMiddle) {
            this.lexer.next()
            const parts = this.parseTemplateParts()
            if (token.value) {
                parts.unshift({ Literal: { type: TokenType.String, value: token.value } })
            }
            return {
                Template: { parts },
            }
        }

        throw new SyntaxError(
            `Unexpected token at beginning of template: ${JSON.stringify(token.value)} (at ${this.lexer.index})`
        )
    }

    // Primary ::= Identifier |
    //             String |
    //             Template |
    //             Number |
    //             FunctionCall
    private parsePrimary(): ExpressionNode {
        const token = this.lexer.peek()
        if (token === undefined) {
            throw new SyntaxError(`Unexpected termination of expression (at ${this.lexer.index})`)
        }

        if (token.type === TokenType.Identifier) {
            this.lexer.next()
            if (matchOp(this.lexer.peek(), '(')) {
                return this.parseFunctionCall(token.value)
            }
            return {
                Identifier: token.value,
            }
        }

        if (token.type === TokenType.String || token.type === TokenType.Number) {
            this.lexer.next()
            return { Literal: { type: token.type, value: token.value } }
        }

        if (
            token.type === TokenType.NoSubstitutionTemplate ||
            token.type === TokenType.TemplateHead ||
            token.type === TokenType.TemplateMiddle
        ) {
            return this.parseTemplate()
        }

        if (matchOp(token, '(')) {
            this.lexer.next()
            const expression = this.parseAdditive()
            const token = this.lexer.next()
            if (!matchOp(token, ')')) {
                throw new SyntaxError(`Expected ")" (at ${this.lexer.index})`)
            }
            return expression
        }

        throw new SyntaxError(`Parse error on token: ${JSON.stringify(token.value)} (at ${this.lexer.index})`)
    }

    // Unary ::= Primary |
    //           UnaryOp Unary
    private parseUnary(): ExpressionNode {
        const token = this.lexer.peek()
        if (token !== undefined && (matchOp(token, '-') || matchOp(token, '+') || matchOp(token, '!'))) {
            this.lexer.next()
            const expression = this.parseUnary()
            return {
                Unary: {
                    operator: token.value,
                    expression,
                },
            }
        }
        return this.parsePrimary()
    }

    // Multiplicative ::= Unary |
    //                    Multiplicative BinaryOp Unary
    private parseMultiplicative(): ExpressionNode {
        let expression = this.parseUnary()
        let token = this.lexer.peek()
        while (token !== undefined && (matchOp(token, '*') || matchOp(token, '/') || matchOp(token, '%'))) {
            this.lexer.next()
            expression = {
                Binary: {
                    operator: token.value,
                    left: expression,
                    right: this.parseUnary(),
                },
            }
            token = this.lexer.peek()
        }
        return expression
    }

    // Additive ::= Multiplicative |
    //              Additive BinaryOp Multiplicative
    private parseAdditive(): ExpressionNode {
        let expression = this.parseMultiplicative()
        let token = this.lexer.peek()
        while (
            token !== undefined &&
            (matchOp(token, '+') ||
                matchOp(token, '-') ||
                matchOp(token, '==') ||
                matchOp(token, '!=') ||
                matchOp(token, '===') ||
                matchOp(token, '!==') ||
                matchOp(token, '<') ||
                matchOp(token, '>') ||
                matchOp(token, '<=') ||
                matchOp(token, '>=') ||
                matchOp(token, '&&') ||
                matchOp(token, '||'))
        ) {
            this.lexer.next()
            expression = {
                Binary: {
                    operator: token.value,
                    left: expression,
                    right: this.parseMultiplicative(),
                },
            }
            token = this.lexer.peek()
        }
        return expression
    }

    // Expression ::= Additive
    private parseExpression(): ExpressionNode {
        return this.parseAdditive()
    }
}

/** Parses a template. */
export class TemplateParser extends Parser {
    public parse(templateString: string): ExpressionNode {
        if (!this.lexer) {
            this.lexer = new TemplateLexer()
        }

        this.lexer.reset(templateString)
        const expression = this.parseTemplate()

        const token = this.lexer.next()
        if (token !== undefined) {
            throw new SyntaxError(
                `Unexpected token at end of template input: ${JSON.stringify(token.value)} (at ${this.lexer.index})`
            )
        }

        return expression
    }
}

function matchOp(token: Pick<Token, 'type' | 'value'> | undefined, operator: Operator): boolean {
    return token !== undefined && token.type === TokenType.Operator && token.value === operator
}
