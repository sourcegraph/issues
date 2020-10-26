import React from 'react'
import * as GQL from '../../../../../shared/src/graphql/schema'

/** Displays the price of a tiered plan. */
export const ProductPlanTiered: React.FunctionComponent<{
    plan: Pick<GQL.IProductPlan, 'planTiers' | 'tiersMode' | 'minQuantity'>
}> = ({ plan: { planTiers, tiersMode, minQuantity } }) => (
    <>
        {planTiers.map((tier, index) => (
            <div key={index}>
                {formatAmountForTier(tier, minQuantity)} {formatLabelForTier(tier, tiersMode, planTiers[index - 1])}
            </div>
        ))}
    </>
)

function formatAmountForTier(tier: GQL.IPlanTier, minQuantity: number | null): string {
    if (minQuantity !== null && tier.upTo !== 0 && tier.upTo <= minQuantity) {
        const amount = tier.flatAmount
            ? tier.flatAmount / 100
            : (tier.unitAmount / 100) /* cents in a USD */ * minQuantity
        // Quote the total annual amount for users up to the minQuantity.
        const localizedAmount = amount.toLocaleString('en-US', {
            style: 'currency',
            currency: 'USD',
            minimumFractionDigits: 0,
        })
        return `${localizedAmount}/year total`
    }

    if (tier.unitAmount === 0) {
        // Quote the flat amount.
        const amount = (tier.flatAmount / 100 /* cents in a USD */ / 12) /* months */
            .toLocaleString('en-US', { style: 'currency', currency: 'USD', minimumFractionDigits: 0 })
        return `${amount}/month`
    }

    // Quote the $/user/month amount.
    const amount = (tier.unitAmount / 100 /* cents in a USD */ / 12) /* months */
        .toLocaleString('en-US', { style: 'currency', currency: 'USD', minimumFractionDigits: 0 })
    return `${amount}/user/month`
}

function formatLabelForTier(tier: GQL.IPlanTier, tiersMode: string, previousTier?: GQL.IPlanTier): string {
    if (tiersMode === 'volume') {
        if (!previousTier) {
            return `for 1–${tier.upTo} users`
        }
        if (tier.upTo === 0) {
            return `for ${previousTier.upTo + 1} or more users`
        }
        return `for ${previousTier.upTo + 1}–${tier.upTo} users`
    }

    if (!previousTier) {
        return `for the first ${tier.upTo} users`
    }
    if (tier.upTo === 0) {
        return 'for each additional user (paid yearly)'
    }
    if (previousTier) {
        return `for the next ${tier.upTo - previousTier.upTo} users`
    }
    return `up to ${tier.upTo} users`
}
