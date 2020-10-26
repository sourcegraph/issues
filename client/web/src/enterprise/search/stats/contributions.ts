import * as H from 'history'
import { Subscription, Unsubscribable } from 'rxjs'
import { parseContributionExpressions } from '../../../../../shared/src/api/client/services/contribution'
import { ContributableMenu } from '../../../../../shared/src/api/protocol'
import { ExtensionsControllerProps } from '../../../../../shared/src/extensions/controller'

export function registerSearchStatsContributions({
    extensionsController,
}: {
    history: H.History
} & ExtensionsControllerProps<'services'>): Unsubscribable {
    const subscriptions = new Subscription()

    const ACTION_ID = 'search.stats'
    subscriptions.add(
        extensionsController.services.contribution.registerContributions({
            contributions: parseContributionExpressions({
                actions: [
                    {
                        id: ACTION_ID,
                        title: 'Statistics',
                        category: 'Search',
                        command: 'open',
                        // eslint-disable-next-line no-template-curly-in-string
                        commandArguments: ['/stats?q=${get(context, "searchQuery")}'],
                        actionItem: {
                            label: 'Stats',
                        },
                    },
                ],
                menus: {
                    [ContributableMenu.SearchResultsToolbar]: [
                        {
                            action: ACTION_ID,
                            when: 'config.experimentalFeatures && get(config.experimentalFeatures, "searchStats")',
                        },
                    ],
                },
            }),
        })
    )

    return subscriptions
}
