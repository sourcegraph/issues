import { noop, take } from 'lodash'
import { from, isObservable } from 'rxjs'
import * as sinon from 'sinon'
import {
    ContributionRegistry,
    ContributionsEntry,
    ContributionUnsubscribable,
} from '../api/client/services/contribution'
import { ExecutableExtension } from '../api/client/services/extensionsService'
import { Contributions } from '../api/protocol/contribution'
import { registerExtensionContributions } from './controller'
import { ExtensionManifest } from './extensionManifest'

describe('registerExtensionContributions()', () => {
    test('registers a contributions observable that emits contributions from all extensions', async () => {
        const registerContributions = sinon.spy(
            (entry: ContributionsEntry): ContributionUnsubscribable => ({ entry, unsubscribe: noop })
        )
        const fakeRegistry: Pick<ContributionRegistry, 'registerContributions'> = {
            registerContributions,
        }
        const extensions: ExecutableExtension[] = [
            {
                id: '1',
                scriptURL: '1.js',
                manifest: {
                    url: '',
                    activationEvents: [],
                    contributes: {
                        configuration: {
                            '1.config': '1',
                        },
                    },
                },
            },
            {
                id: '2',
                scriptURL: '2.js',
                manifest: {
                    url: '',
                    activationEvents: [],
                    contributes: {
                        configuration: {
                            '2.config': '2',
                        },
                    },
                },
            },
            {
                id: '3',
                scriptURL: '3.js',
                manifest: {
                    url: '',
                    activationEvents: [],
                    contributes: {
                        configuration: {
                            '3.config': '3',
                        },
                    },
                },
            },
        ]
        registerExtensionContributions(fakeRegistry, {
            activeExtensions: from([take(extensions, 1), take(extensions, 2), extensions]),
        })
        sinon.assert.calledOnce(registerContributions)
        const extensionContributions = registerContributions.getCalls()[0].args[0].contributions
        if (!isObservable<Contributions[]>(extensionContributions)) {
            throw new Error('Expected extensionContributions to be Observable')
        }
        expect(await extensionContributions.toPromise()).toEqual(
            extensions.map(({ manifest }) => (manifest as NonNullable<ExtensionManifest>).contributes)
        )
    })
})
