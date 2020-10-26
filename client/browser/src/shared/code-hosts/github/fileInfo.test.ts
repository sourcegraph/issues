import { readFile } from 'mz/fs'
import { getFilePath } from './util'

const tests = [
    ['github.com/blob/vanilla/page.html', 'shared/src/api/extension/types/url.ts'],
    ['github.com/blob/refined-github/page.html', 'shared/src/api/extension/types/url.ts'],
    ['ghe-2.14.11/blob/vanilla/page.html', 'bench_test.go'],
    ['ghe-2.14.11/blob/refined-github/page.html', 'bench_test.go'],
]

describe('github/fileInfo', () => {
    describe('getFilePath()', () => {
        for (const [fixture, expectedFilePath] of tests) {
            it(`finds the file path in ${fixture}`, async () => {
                document.body.innerHTML = await readFile(`${__dirname}/__fixtures__/${fixture}`, 'utf-8')
                expect(getFilePath()).toBe(expectedFilePath)
            })
        }
    })
})
