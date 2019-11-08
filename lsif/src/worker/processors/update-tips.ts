import { ConfigurationFetcher } from '../../config'
import { TracingContext } from '../../tracing'
import { XrepoDatabase } from '../../xrepo'

/*
 * Create a job that updates the tip of the default branch for every repository that has LSIF data.
 *
 * @param xrepoDatabase The cross-repo database.
 * @param fetchConfiguration A function that returns the current configuration.
 */
export const createUpdateTipsJobProcessor = (
    xrepoDatabase: XrepoDatabase,
    fetchConfiguration: ConfigurationFetcher
) => (args: { [K: string]: any }, ctx: TracingContext): Promise<void> =>
    xrepoDatabase.discoverAndUpdateTips({
        gitserverUrls: fetchConfiguration().gitServers,
        ctx,
    })
