import { Unsubscribable } from 'rxjs'
import { parseContributionExpressions } from '../../../../../shared/src/api/client/services/contribution'
import { ExtensionsControllerProps } from '../../../../../shared/src/extensions/controller'

export function registerPanelToolbarContributions(
    contributionService: Pick<
        ExtensionsControllerProps['extensionsController']['services']['contribution'],
        'registerContributions'
    >
): Unsubscribable {
    return contributionService.registerContributions({
        contributions: parseContributionExpressions({
            actions: [
                {
                    id: 'panel.locations.groupByFile',
                    title: 'Group by file',
                    category: 'Locations (panel)',
                    command: 'updateConfiguration',
                    commandArguments: [
                        ['panel.locations.groupByFile'],
                        // eslint-disable-next-line no-template-curly-in-string
                        '${!config.panel.locations.groupByFile}',
                        null,
                        'json',
                    ],
                    // eslint-disable-next-line no-template-curly-in-string
                    actionItem: { label: '${config.panel.locations.groupByFile && "Ungroup" || "Group"} by file' },
                },
            ],
            menus: {
                'panel/toolbar': [
                    {
                        action: 'panel.locations.groupByFile',
                        when: 'panel.locations.hasResults && panel.activeView.hasLocations',
                    },
                ],
            },
        }),
    })
}
