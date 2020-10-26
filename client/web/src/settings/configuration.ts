import settingsSchemaJSON from '../../../../schema/settings.schema.json'
import { ConfiguredRegistryExtension } from '../../../shared/src/extensions/extension'
import { isErrorLike } from '../../../shared/src/util/errors'

/**
 * Merges settings schemas from base settings and all configured extensions.
 *
 * @param configuredExtensions
 * @returns A JSON Schema that describes an instance of settings for a particular subject.
 */
export function mergeSettingsSchemas(configuredExtensions: Pick<ConfiguredRegistryExtension, 'manifest'>[]): any {
    return {
        allOf: [
            { $ref: settingsSchemaJSON.$id },
            ...(configuredExtensions || [])
                .map(configuredExtension => {
                    if (
                        configuredExtension.manifest &&
                        !isErrorLike(configuredExtension.manifest) &&
                        configuredExtension.manifest.contributes?.configuration
                    ) {
                        // Adjust the schema to describe a valid instance of settings for a subject (instead of the
                        // final, merged settings).
                        //
                        // This is necessary to avoid erroneous validation errors. For example, suppose an extension's
                        // configuration schema declares that the property "x" is required. For the configuration to be
                        // valid, "x" may be set in global, organization, or user settings. It is valid for user
                        // settings to NOT contain "x" (if global or organization settings contains "x").
                        //
                        // The JSON Schema returned by mergeSettingsSchema is used for a single subject's settings
                        // (e.g., for user settings in the above example). Therefore, we must allow additionalProperties
                        // and set required to [] to avoid erroneous validation errors.
                        return {
                            ...configuredExtension.manifest.contributes.configuration,

                            // Force allow additionalProperties to prevent any single extension's configuration schema
                            // from invalidating all other extensions' configuration properties.
                            additionalProperties: true,

                            // Force no required properties because this instance is only the settings for a single
                            // subject. It is possible that a required property is specified at a different subject in
                            // the cascade, in which case we don't want to report this instance as invalid.
                            required: [],
                        }
                    }
                    return true // JSON Schema that matches everything
                })
                .filter(schema => schema !== true), // omit trivial JSON Schemas
        ],
    }
}
