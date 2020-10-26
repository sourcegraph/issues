import { uniqueId, noop } from 'lodash'
import { from, NEVER, Subject, Subscription } from 'rxjs'
import { first } from 'rxjs/operators'
import { Services } from '../../../../../shared/src/api/client/services'
import { CodeEditor } from '../../../../../shared/src/api/client/services/viewerService'
import { integrationTestContext } from '../../../../../shared/src/api/integration-test/testHelpers'
import { Controller } from '../../../../../shared/src/extensions/controller'
import { MutationRecordLike } from '../../util/dom'
import { handleTextFields } from './textFields'
import { pretendRemote } from '../../../../../shared/src/api/util'
import { FlatExtensionHostAPI } from '../../../../../shared/src/api/contract'

jest.mock('uuid', () => ({
    v4: () => 'uuid',
}))

const createMockController = (services: Services): Controller => ({
    services,
    notifications: NEVER,
    executeCommand: () => Promise.resolve(),
    unsubscribe: noop,
    extHostAPI: Promise.resolve(pretendRemote<FlatExtensionHostAPI>({})),
})

describe('textFields', () => {
    beforeEach(() => {
        document.body.innerHTML = ''
    })

    describe('handleTextFields()', () => {
        let subscriptions = new Subscription()

        afterEach(() => {
            subscriptions.unsubscribe()
            subscriptions = new Subscription()
        })

        const createTestElement = (): HTMLTextAreaElement => {
            const element = document.createElement('textarea')
            element.className = `test test-${uniqueId()}`
            document.body.append(element)
            return element
        }

        test('detects addition and removal of text fields', async () => {
            const { services } = await integrationTestContext(undefined, { roots: [], viewers: [] })
            const textFieldElement = createTestElement()
            textFieldElement.id = 'text-field'
            textFieldElement.value = 'abc'
            textFieldElement.setSelectionRange(2, 3)

            const mutations = new Subject<MutationRecordLike[]>()

            subscriptions.add(
                handleTextFields(
                    mutations,
                    { extensionsController: createMockController(services) },
                    {
                        textFieldResolvers: [
                            { selector: 'textarea', resolveView: () => ({ element: textFieldElement }) },
                        ],
                    }
                )
            )

            // Add text field.
            mutations.next([{ addedNodes: [document.body], removedNodes: [] }])
            await from(services.viewer.viewerUpdates).pipe(first()).toPromise()
            expect([...services.viewer.viewers.values()]).toEqual([
                {
                    viewerId: 'viewer#0',
                    isActive: true,
                    resource: 'comment://0',
                    selections: [
                        {
                            anchor: { line: 0, character: 2 },
                            active: { line: 0, character: 3 },
                            start: { line: 0, character: 2 },
                            end: { line: 0, character: 3 },
                            isReversed: false,
                        },
                    ],
                    type: 'CodeEditor',
                },
            ] as CodeEditor[])

            // Remove text field.
            textFieldElement.remove()
            mutations.next([{ addedNodes: [], removedNodes: [textFieldElement] }])
            await from(services.viewer.viewerUpdates).pipe(first()).toPromise()
            expect(services.viewer.viewers.size).toEqual(0)
        })
    })
})
