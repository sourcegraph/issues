import { camelCase } from 'lodash'

import { InsightType, InsightTypePrefix, LangStatsInsight } from '../../../../../core/types'
import { LangStatsCreationFormFields } from '../types'

/**
 * Converter from creation UI form values to real insight object.
 * */
export const getSanitizedLangStatsInsight = (values: LangStatsCreationFormFields): LangStatsInsight => ({
    type: InsightType.Extension,
    // ID generated according to our naming insight convention
    // <Type of insight>.insight.<name of insight>
    id: `${InsightTypePrefix.langStats}.${camelCase(values.title)}`,
    visibility: values.visibility,
    title: values.title.trim(),
    repository: values.repository.trim(),
    otherThreshold: values.threshold / 100,
})
