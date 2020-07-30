import { ProductSubscriptionsResult } from '../../../graphql-operations'

export type GraphQlProductSubscriptionConnection = ProductSubscriptionsResult['dotcom']['productSubscriptions']
export type GraphQlProductSubscriptionNode = GraphQlProductSubscriptionConnection['nodes'][number]
