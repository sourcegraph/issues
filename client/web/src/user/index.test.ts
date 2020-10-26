import { VALID_USERNAME_REGEXP } from '.'

describe('VALID_USERNAME_REGEX', () => {
    const VALID_USERNAMES: string[] = [
        'fo',
        'foo',
        'FoO',
        'FOO',
        '1foo',
        'foo1',
        'Fo1o',
        'foo-bar',
        'foo.bar',
        'foo-32',
        '32-foo',
        'foo-bar-',
        '42',
    ]

    for (const username of VALID_USERNAMES) {
        test(`should match ${JSON.stringify(username)}`, () => {
            expect(username.match(VALID_USERNAME_REGEXP)).toBeTruthy()
        })
    }

    const INVALID_USERNAMES: string[] = ['!foo', '-foo', 'foo_bar']

    for (const username of INVALID_USERNAMES) {
        test(`should not match ${JSON.stringify(username)}`, () => {
            expect(username.match(VALID_USERNAME_REGEXP)).toBeNull()
        })
    }
})
