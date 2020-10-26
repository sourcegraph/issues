import { Context } from '../context'
import { parse, parseTemplate } from './evaluator'

const FIXTURE_CONTEXT: Context = {
    a: 1,
    b: 1,
    c: 2,
    x: 'y',
    o: { k: 'v' },
    array: [7],
}

describe('Expression', () => {
    /* eslint-disable no-template-curly-in-string */
    const TESTS = {
        a: 1,
        'a + b': 2,
        'a == b': true,
        'a != b': false,
        'a + b == c': true,
        x: 'y',
        'd === false': false,
        'd !== false': true,
        '!a': false,
        '!!a': true,
        'a && c': 2,
        'a || b': 1,
        '(a + b) * 2': 4,
        'x == "y"': true,
        'json(o)': '{"k":"v"}',
        // TODO: Support operator precedence. See ./parser.test.ts for a commented-out precedence test case.
        //
        // 'x == "y" || x == "z"': true,
        'x == "y" && x == "z"': false,
        'x == "y" && x != "z"': true,
        '`a`': 'a',
        '`${x}`': 'y',
        '`a${x}b`': 'ayb',
        '`_${x}_${a}_${a+b}`': '_y_1_2',
        '`_${`-${x}-`}_`': '_-y-_',
        'a || isnotdefined': 1, // short-circuit (if not, the use of an undefined ident would cause an error)
        'get(array, 0)': 7,
        'get(array, 1)': undefined, // out-of-bounds array index is undefined
        'get(context, "c")': 2,
    }
    /* eslint-enable no-template-curly-in-string */
    for (const [expression, want] of Object.entries(TESTS)) {
        test(expression, () => {
            const value = parse<unknown>(expression).exec(FIXTURE_CONTEXT)
            expect(value).toBe(want)
        })
    }
})

describe('TemplateExpression', () => {
    /* eslint-disable no-template-curly-in-string */
    const TESTS = {
        a: 'a',
        '${x}': 'y',
        'a${x}b': 'ayb',
        '_${x}_${a}_${a+b}': '_y_1_2',
        '_${`-${x}-`}_': '_-y-_',
    }
    /* eslint-enable no-template-curly-in-string */
    for (const [template, want] of Object.entries(TESTS)) {
        test(template, () => {
            const value = parseTemplate(template).exec(FIXTURE_CONTEXT)
            expect(value).toBe(want)
        })
    }
})
