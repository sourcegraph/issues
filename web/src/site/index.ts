import * as GQL from '../../../shared/src/graphql/schema'

export type SiteFlags = Pick<
    GQL.Site,
    | 'needsRepositoryConfiguration'
    | 'freeUsersExceeded'
    | 'alerts'
    | 'authProviders'
    | 'disableBuiltInSearches'
    | 'sendsEmailVerificationEmails'
    | 'updateCheck'
    | 'productSubscription'
    | 'productVersion'
>
