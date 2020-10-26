import { ExtensionCategory, EXTENSION_CATEGORIES } from '../../../shared/src/schema/extensionSchema'
import { Settings } from '../../../shared/src/settings/settings'
import { createRecord } from '../../../shared/src/util/createRecord'
import { applyCategoryFilter, applyExtensionsEnablement } from './extensions'

describe('extension registry helpers', () => {
    const TEST_EXTENSION_CATEGORIES: ExtensionCategory[] = ['Reports and stats', 'Linters']

    const extensionsByCategory: Parameters<typeof applyCategoryFilter>[0] = createRecord(
        EXTENSION_CATEGORIES,
        category => {
            let primaryExtensionIDs: string[] = []
            let allExtensionIDs: string[] = []

            if (category === 'Reports and stats') {
                primaryExtensionIDs = ['sourcegraph/codecov']
                allExtensionIDs = ['sourcegraph/codecov', 'sourcegraph/eslint']
            }

            if (category === 'Linters') {
                primaryExtensionIDs = ['sourcegraph/eslint', 'sourcegraph/dockerfile-lint']
                allExtensionIDs = ['sourcegraph/eslint', 'sourcegraph/dockerfile-lint']
            }

            return {
                primaryExtensionIDs,
                allExtensionIDs,
            }
        }
    )

    describe('applyCategoryFilter', () => {
        test('no selected categories', () => {
            const filteredExtensionsByCategory = applyCategoryFilter(
                extensionsByCategory,
                TEST_EXTENSION_CATEGORIES,
                []
            )

            // Should be primary IDs of each base category provided
            expect(filteredExtensionsByCategory).toStrictEqual({
                'Reports and stats': extensionsByCategory['Reports and stats'].primaryExtensionIDs,
                Linters: extensionsByCategory.Linters.primaryExtensionIDs,
            })
        })

        test('one selected category', () => {
            const filteredExtensionsByCategory = applyCategoryFilter(extensionsByCategory, TEST_EXTENSION_CATEGORIES, [
                'Reports and stats',
            ])

            // Should include all IDs of the selected category
            expect(filteredExtensionsByCategory).toStrictEqual({
                'Reports and stats': extensionsByCategory['Reports and stats'].allExtensionIDs,
            })
        })
        test('multiple selected categories', () => {
            const filteredExtensionsByCategory = applyCategoryFilter(extensionsByCategory, TEST_EXTENSION_CATEGORIES, [
                'Reports and stats',
                'Linters',
            ])

            expect(filteredExtensionsByCategory).toStrictEqual({
                'Reports and stats': extensionsByCategory['Reports and stats'].allExtensionIDs,
                // Every linter extension that isn't in reports and stats
                Linters: extensionsByCategory.Linters.allExtensionIDs.filter(
                    extensionID => !extensionsByCategory['Reports and stats'].allExtensionIDs.includes(extensionID)
                ),
            })

            // reverse order
            const filteredExtensionsByCategoryReverseOrder = applyCategoryFilter(
                extensionsByCategory,
                TEST_EXTENSION_CATEGORIES,
                ['Linters', 'Reports and stats']
            )

            expect(filteredExtensionsByCategoryReverseOrder).toStrictEqual({
                // Every reports and stats extension that isn't in linters
                'Reports and stats': extensionsByCategory['Reports and stats'].allExtensionIDs.filter(
                    extensionID => !extensionsByCategory.Linters.allExtensionIDs.includes(extensionID)
                ),
                Linters: extensionsByCategory.Linters.allExtensionIDs,
            })
        })
    })

    describe('applyExtensionsEnablement', () => {
        const MINIMAL_SETTINGS: Settings = {
            extensions: {
                'sourcegraph/codecov': true,
                'sourcegraph/eslint': true,
                'sourcegraph/dockerfile-lint': false,
            },
        }

        const categorizedExtensions: Parameters<typeof applyExtensionsEnablement>[0] = createRecord(
            TEST_EXTENSION_CATEGORIES,
            category => {
                if (category === 'Reports and stats') {
                    return ['sourcegraph/codecov']
                }

                if (category === 'Linters') {
                    return ['sourcegraph/eslint', 'sourcegraph/dockerfile-lint']
                }

                return []
            }
        )

        test('all', () => {
            const allExtensions = applyExtensionsEnablement(
                categorizedExtensions,
                TEST_EXTENSION_CATEGORIES,
                'all',
                MINIMAL_SETTINGS
            )

            expect(allExtensions).toStrictEqual(categorizedExtensions)
        })
        test('enabled', () => {
            const enabledExtensions = applyExtensionsEnablement(
                categorizedExtensions,
                TEST_EXTENSION_CATEGORIES,
                'enabled',
                MINIMAL_SETTINGS
            )

            expect(enabledExtensions).toStrictEqual({
                'Reports and stats': ['sourcegraph/codecov'],
                Linters: ['sourcegraph/eslint'],
            })
        })
        test('disabled', () => {
            const disabledExtensions = applyExtensionsEnablement(
                categorizedExtensions,
                TEST_EXTENSION_CATEGORIES,
                'disabled',
                MINIMAL_SETTINGS
            )

            expect(disabledExtensions).toStrictEqual({
                'Reports and stats': [],
                Linters: ['sourcegraph/dockerfile-lint'],
            })
        })
    })
})
