import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { gql, dataOrThrowErrors } from '../../../../../shared/src/graphql/graphql'
import { queryGraphQL } from '../../../backend/graphql'
import { ProductPlansResult } from '../../../graphql-operations'

export type ProductPlan = ProductPlansResult['dotcom']['productPlans'][number]

export function queryProductPlans(): Observable<ProductPlan[]> {
    return queryGraphQL<ProductPlansResult>(
        gql`
            query ProductPlans {
                dotcom {
                    productPlans {
                        productPlanID
                        billingPlanID
                        name
                        pricePerUserPerYear
                        minQuantity
                        maxQuantity
                        tiersMode
                        planTiers {
                            unitAmount
                            upTo
                            flatAmount
                        }
                    }
                }
            }
        `
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.dotcom.productPlans)
    )
}
