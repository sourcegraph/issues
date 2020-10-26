import { ActionItemAction } from '../actions/ActionItem'
import { ContributableMenu } from '../api/protocol'
import { getContributedActionItems } from './contributions'

describe('getContributedActionItems', () => {
    test('gets action items', () =>
        expect(
            getContributedActionItems(
                {
                    actions: [
                        { id: 'a', command: 'a', title: 'ta', description: 'da' },
                        { id: 'b', command: 'b', title: 'tb', description: 'db' },
                        { id: 'c', command: 'c', title: 'tc', description: 'dc' },
                    ],
                    menus: {
                        commandPalette: [
                            { action: 'a', group: '2' },
                            { action: 'b', group: '1', alt: 'c' },
                        ],
                        'editor/title': [{ action: 'c' }],
                    },
                },
                ContributableMenu.CommandPalette
            )
        ).toEqual([
            {
                action: { id: 'b', command: 'b', title: 'tb', description: 'db' },
                altAction: { id: 'c', command: 'c', title: 'tc', description: 'dc' },
            },
            { action: { id: 'a', command: 'a', title: 'ta', description: 'da' }, altAction: undefined },
        ] as ActionItemAction[]))
})
