import { createRecord } from './createRecord'

describe('createRecord', () => {
    it('creates an object with the proper keys and initial values', () => {
        const fruits = ['apple', 'pear', 'kiwi', 'kiwi'] as const
        const record = createRecord(fruits, key => [`first ${key}`])
        expect(record).toStrictEqual({
            apple: ['first apple'],
            pear: ['first pear'],
            kiwi: ['first kiwi'],
        })

        record.apple.push('second apple')
        expect(record).toStrictEqual({
            apple: ['first apple', 'second apple'],
            pear: ['first pear'],
            kiwi: ['first kiwi'],
        })
    })
})
