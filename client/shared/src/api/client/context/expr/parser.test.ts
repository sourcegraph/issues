import { TokenType } from './lexer'
import { ExpressionNode, Parser, TemplateParser } from './parser'

describe('Parser', () => {
    const TESTS: { [expr: string]: ExpressionNode } = {
        '!a': {
            Unary: {
                operator: '!',
                expression: { Identifier: 'a' },
            },
        },
        '"a"': {
            Literal: {
                type: TokenType.String,
                value: 'a',
            },
        },
        '`a`': {
            Literal: {
                type: TokenType.String,
                value: 'a',
            },
        },
        // eslint-disable-next-line no-template-curly-in-string
        '`${x}`': {
            Template: {
                parts: [{ Identifier: 'x' }],
            },
        },
        // eslint-disable-next-line no-template-curly-in-string
        '`${x}${y}`': {
            Template: {
                parts: [{ Identifier: 'x' }, { Identifier: 'y' }],
            },
        },
        // eslint-disable-next-line no-template-curly-in-string
        '`${x+y}`': {
            Template: {
                parts: [
                    {
                        Binary: {
                            left: { Identifier: 'x' },
                            operator: '+',
                            right: { Identifier: 'y' },
                        },
                    },
                ],
            },
        },
        // eslint-disable-next-line no-template-curly-in-string
        '`a${x}`': {
            Template: {
                parts: [{ Literal: { type: TokenType.String, value: 'a' } }, { Identifier: 'x' }],
            },
        },
        // eslint-disable-next-line no-template-curly-in-string
        '`${x}b`': {
            Template: {
                parts: [{ Identifier: 'x' }, { Literal: { type: TokenType.String, value: 'b' } }],
            },
        },
        // eslint-disable-next-line no-template-curly-in-string
        '`a${x}b`': {
            Template: {
                parts: [
                    { Literal: { type: TokenType.String, value: 'a' } },
                    { Identifier: 'x' },
                    { Literal: { type: TokenType.String, value: 'b' } },
                ],
            },
        },
        // eslint-disable-next-line no-template-curly-in-string
        '`a${x}b${y}c`': {
            Template: {
                parts: [
                    { Literal: { type: TokenType.String, value: 'a' } },
                    { Identifier: 'x' },
                    { Literal: { type: TokenType.String, value: 'b' } },
                    { Identifier: 'y' },
                    { Literal: { type: TokenType.String, value: 'c' } },
                ],
            },
        },
        // eslint-disable-next-line no-template-curly-in-string
        '`a${`x${y}z`}b`': {
            Template: {
                parts: [
                    { Literal: { type: TokenType.String, value: 'a' } },
                    {
                        Template: {
                            parts: [
                                { Literal: { type: TokenType.String, value: 'x' } },
                                { Identifier: 'y' },
                                { Literal: { type: TokenType.String, value: 'z' } },
                            ],
                        },
                    },
                    { Literal: { type: TokenType.String, value: 'b' } },
                ],
            },
        },
        'a && b': {
            Binary: {
                left: { Identifier: 'a' },
                operator: '&&',
                right: { Identifier: 'b' },
            },
        },

        // TODO: The template language currently does not support operator precedence. You must use parentheses to
        // be explicit. This commented-out (failing) test case is the desired parse tree for this expression:
        //
        // 'a == b && c == d': {
        //     Binary: {
        //         left: {
        //             Binary: {
        //                 left: { Identifier: 'a' },
        //                 operator: '==',
        //                 right: {
        //                     Identifier: 'b',
        //                 },
        //             },
        //         },
        //         operator: '&&',
        //         right: {
        //             Binary: {
        //                 left: { Identifier: 'c' },
        //                 operator: '==',
        //                 right: { Identifier: 'd' },
        //             },
        //         },
        //     },
        // },
        //
        // This is the undesirable parse tree for the expression. When the commented-out test case passes, this
        // undesirable test case should be removed.
        'a == b && c == d': {
            Binary: {
                left: {
                    Binary: {
                        left: {
                            Binary: {
                                left: { Identifier: 'a' },
                                operator: '==',
                                right: { Identifier: 'b' },
                            },
                        },
                        operator: '&&',
                        right: {
                            Identifier: 'c',
                        },
                    },
                },
                operator: '==',
                right: {
                    Identifier: 'd',
                },
            },
        },

        '(a + b) * c': {
            Binary: {
                left: {
                    Binary: {
                        left: { Identifier: 'a' },
                        operator: '+',
                        right: { Identifier: 'b' },
                    },
                },
                operator: '*',
                right: { Identifier: 'c' },
            },
        },
        'ab * 1 + x(2, 3)': {
            Binary: {
                left: {
                    Binary: {
                        left: { Identifier: 'ab' },
                        operator: '*',
                        right: { Literal: { type: TokenType.Number, value: '1' } },
                    },
                },
                operator: '+',
                right: {
                    FunctionCall: {
                        name: 'x',
                        args: [
                            { Literal: { type: TokenType.Number, value: '2' } },
                            { Literal: { type: TokenType.Number, value: '3' } },
                        ],
                    },
                },
            },
        },
    }
    const parser = new Parser()
    for (const [expression, want] of Object.entries(TESTS)) {
        test(expression, () => expect(parser.parse(expression)).toEqual(want))
    }

    test('throws an error on an invalid argument list', () => expect(() => parser.parse('a(1,,)')).toThrow())
    test('throws an error on an unclosed string', () => expect(() => parser.parse('"')).toThrow())
    test('throws an error on an unclosed template', () => expect(() => parser.parse('`')).toThrow())
    test('throws an error on an invalid unary operator', () => expect(() => parser.parse('!')).toThrow())
    test('throws an error on an invalid binary operator', () => expect(() => parser.parse('a*')).toThrow())
    test('throws an error on an unclosed function call', () => expect(() => parser.parse('a(')).toThrow())
    test('throws an error on an unterminated expression', () => expect(() => parser.parse('(a=')).toThrow())
})

describe('TemplateParser', () => {
    const TESTS: { [template: string]: ExpressionNode } = {
        // eslint-disable-next-line no-template-curly-in-string
        '${x}': {
            Template: {
                parts: [{ Identifier: 'x' }],
            },
        },
        // eslint-disable-next-line no-template-curly-in-string
        '${x}${y}': {
            Template: {
                parts: [{ Identifier: 'x' }, { Identifier: 'y' }],
            },
        },
        // eslint-disable-next-line no-template-curly-in-string
        '${x+y}': {
            Template: {
                parts: [
                    {
                        Binary: {
                            left: { Identifier: 'x' },
                            operator: '+',
                            right: { Identifier: 'y' },
                        },
                    },
                ],
            },
        },
        // eslint-disable-next-line no-template-curly-in-string
        'a${x}': {
            Template: {
                parts: [{ Literal: { type: TokenType.String, value: 'a' } }, { Identifier: 'x' }],
            },
        },
        // eslint-disable-next-line no-template-curly-in-string
        '${x}b': {
            Template: {
                parts: [{ Identifier: 'x' }, { Literal: { type: TokenType.String, value: 'b' } }],
            },
        },
        // eslint-disable-next-line no-template-curly-in-string
        'a${x}b': {
            Template: {
                parts: [
                    { Literal: { type: TokenType.String, value: 'a' } },
                    { Identifier: 'x' },
                    { Literal: { type: TokenType.String, value: 'b' } },
                ],
            },
        },
        // eslint-disable-next-line no-template-curly-in-string
        'a${x}b${y}c': {
            Template: {
                parts: [
                    { Literal: { type: TokenType.String, value: 'a' } },
                    { Identifier: 'x' },
                    { Literal: { type: TokenType.String, value: 'b' } },
                    { Identifier: 'y' },
                    { Literal: { type: TokenType.String, value: 'c' } },
                ],
            },
        },
        // eslint-disable-next-line no-template-curly-in-string
        'a${`x${y}z`}b': {
            Template: {
                parts: [
                    { Literal: { type: TokenType.String, value: 'a' } },
                    {
                        Template: {
                            parts: [
                                { Literal: { type: TokenType.String, value: 'x' } },
                                { Identifier: 'y' },
                                { Literal: { type: TokenType.String, value: 'z' } },
                            ],
                        },
                    },
                    { Literal: { type: TokenType.String, value: 'b' } },
                ],
            },
        },
    }
    const parser = new TemplateParser()
    for (const [template, want] of Object.entries(TESTS)) {
        test(template, () => expect(parser.parse(template)).toEqual(want))
    }
})
