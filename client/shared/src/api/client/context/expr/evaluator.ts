import { Context } from '../context'
import { TokenType } from './lexer'
import { ExpressionNode, Parser, TemplateParser } from './parser'

/**
 * A parsed context expression (that can evaluate to anything)
 */
export class Expression<T> {
    constructor(private root: ExpressionNode) {}

    public exec<C>(context: Context<C>): T {
        return exec<C>(this.root, context)
    }
}

/**
 * A parsed template string that can contain `${contextKey}` interpolations.
 * Always evaluates to a string.
 */
export class TemplateExpression<S extends string = string> extends Expression<S> {}

/**
 * Evaluates an expression with the given context and returns the result.
 */
export function parse<T>(expression: string): Expression<T> {
    return new Expression<T>(new Parser().parse(expression))
}

/**
 * Evaluates a template with the given context and returns the result.
 *
 * A template is a string that interpolates expressions in ${...}. It uses the same syntax as
 * JavaScript templates.
 */
export function parseTemplate(template: string): TemplateExpression {
    return new TemplateExpression(new TemplateParser().parse(template))
}

const FUNCS: { [name: string]: (...args: any[]) => any } = {
    get: (object: any, key: string): any => object?.[key] ?? undefined,
    json: (object: any): string => JSON.stringify(object),
}

function exec<C>(node: ExpressionNode, context: Context<C>): any {
    if ('Literal' in node) {
        switch (node.Literal.type) {
            case TokenType.String:
                return node.Literal.value
            case TokenType.Number:
                return parseFloat(node.Literal.value)
            default:
                throw new SyntaxError(`Unrecognized literal of type ${TokenType[node.Literal.type]}`)
        }
    }

    if ('Template' in node) {
        const parts: any[] = []
        for (const expression of node.Template.parts) {
            parts.push(exec(expression, context))
        }
        return parts.join('')
    }

    if ('Binary' in node) {
        const left = exec(node.Binary.left, context)
        const right = (): any => exec(node.Binary.right, context) // lazy evaluation
        switch (node.Binary.operator) {
            case '&&':
                return left && right()
            case '||':
                return left || right()
            case '==':
                // eslint-disable-next-line eqeqeq
                return left == right()
            case '!=':
                // eslint-disable-next-line eqeqeq
                return left != right()
            case '===':
                return left === right()
            case '!==':
                return left !== right()
            case '<':
                return left < right()
            case '>':
                return left > right()
            case '<=':
                return left <= right()
            case '>=':
                return left >= right()
            case '+':
                // eslint-disable-next-line @typescript-eslint/restrict-plus-operands
                return left + right()
            case '-':
                return left - right()
            case '*':
                return left * right()
            case '/':
                return left / right()
            case '^':
                return left ^ right()
            case '%':
                return left % right()
            default:
                throw new SyntaxError(`Invalid operator: ${node.Binary.operator}`)
        }
    }

    if ('Unary' in node) {
        const expression = exec(node.Unary.expression, context)
        switch (node.Unary.operator) {
            case '!':
                return !expression
            case '+':
                return expression
            case '-':
                return -expression
            default:
                throw new SyntaxError(`Invalid operator: ${node.Unary.operator}`)
        }
    }

    if ('Identifier' in node) {
        switch (node.Identifier) {
            case 'true':
                return true
            case 'false':
                return false
            case 'undefined':
                return undefined
            case 'null':
                return null
            case 'context':
                return context
        }
        return context[node.Identifier]
    }

    if ('FunctionCall' in node) {
        const expression = node.FunctionCall
        const func = FUNCS[expression.name]
        if (typeof func === 'function') {
            const args = expression.args.map(argument => exec(argument, context))
            return func(...args)
        }
        throw new SyntaxError(`Undefined function: ${expression.name}`)
    }

    throw new SyntaxError('Unrecognized syntax node')
}
