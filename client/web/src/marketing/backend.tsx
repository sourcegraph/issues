import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { dataOrThrowErrors, gql } from '../../../shared/src/graphql/graphql'
import * as GQL from '../../../shared/src/graphql/schema'
import { mutateGraphQL, queryGraphQL } from '../backend/graphql'
import { SurveyResponse } from './SurveyPage'
import { UserActivePeriod } from '../graphql-operations'

/**
 * Submits a user satisfaction survey.
 */
export function submitSurvey(input: SurveyResponse): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation SubmitSurvey($input: SurveySubmissionInput!) {
                submitSurvey(input: $input) {
                    alwaysNil
                }
            }
        `,
        { input }
    ).pipe(
        map(dataOrThrowErrors),
        map(() => undefined)
    )
}

/**
 * Fetches survey responses.
 */
export function fetchAllSurveyResponses(args: { first?: number }): Observable<GQL.ISurveyResponseConnection> {
    return queryGraphQL(
        gql`
            query FetchSurveyResponses {
                surveyResponses {
                    nodes {
                        user {
                            id
                            username
                            emails {
                                email
                            }
                        }
                        email
                        score
                        reason
                        better
                        createdAt
                    }
                    totalCount
                }
            }
        `,
        { ...args }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.surveyResponses)
    )
}

/**
 * Fetches users, with their survey responses.
 */
export function fetchAllUsersWithSurveyResponses(args: {
    activePeriod?: UserActivePeriod
    first?: number
    query?: string
}): Observable<GQL.IUserConnection> {
    return queryGraphQL(
        gql`
            query FetchAllUsersWithSurveyResponses($activePeriod: UserActivePeriod, $first: Int, $query: String) {
                users(activePeriod: $activePeriod, first: $first, query: $query) {
                    nodes {
                        id
                        username
                        emails {
                            email
                        }
                        surveyResponses {
                            score
                            reason
                            better
                            createdAt
                        }
                        usageStatistics {
                            lastActiveTime
                        }
                    }
                    totalCount
                }
            }
        `,
        { ...args }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.users)
    )
}

export type SurveyResponseConnectionAggregates = Exclude<GQL.ISurveyResponseConnection, 'nodes'>

/**
 * Fetches survey response aggregate data.
 */
export function fetchSurveyResponseAggregates(): Observable<SurveyResponseConnectionAggregates> {
    return queryGraphQL(
        gql`
            query FetchSurveyResponseAggregates {
                surveyResponses {
                    totalCount
                    last30DaysCount
                    averageScore
                    netPromoterScore
                }
            }
        `
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.surveyResponses)
    )
}

/**
 * Submits a request for a Sourcegraph Enterprise trial license.
 */
export const submitTrialRequest = (email: string): void => {
    mutateGraphQL(
        gql`
            mutation RequestTrial($email: String!) {
                requestTrial(email: $email) {
                    alwaysNil
                }
            }
        `,
        { email }
    )
        .pipe(
            map(dataOrThrowErrors),
            map(() => undefined)
        )
        // eslint-disable-next-line rxjs/no-ignored-subscription
        .subscribe({
            error: () => {
                // Swallow errors since the form submission is a non-blocking request that happens during site-init
                // if a trial license key is requested.
            },
        })
}
