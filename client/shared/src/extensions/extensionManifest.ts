import { isPlainObject } from 'lodash'
import { ExtensionManifest as ExtensionManifestSchema } from '../schema/extensionSchema'
import { ErrorLike, isErrorLike } from '../util/errors'
import { parseJSONCOrError } from '../util/jsonc'

/**
 * Represents an input object that is validated against a subset of properties of the {@link ExtensionManifest}
 * JSON Schema. For simplicity, only necessary fields are validated and included here.
 */
export type ExtensionManifest = Pick<
    ExtensionManifestSchema,
    | 'description'
    | 'repository'
    | 'categories'
    | 'tags'
    | 'readme'
    | 'url'
    | 'icon'
    | 'activationEvents'
    | 'contributes'
    | 'publisher'
>

/**
 * Parses and validates the extension manifest. If parsing or validation fails, an error value is returned (not
 * thrown).
 *
 * @todo Contribution ("contributes" property) validation is incomplete.
 */
export function parseExtensionManifestOrError(input: string): ExtensionManifest | ErrorLike {
    const value = parseJSONCOrError<ExtensionManifest>(input)
    if (!isErrorLike(value)) {
        if (!isPlainObject(value)) {
            return new Error('invalid extension manifest: must be a JSON object')
        }
        const problems: string[] = []
        if (value.repository) {
            if (!isPlainObject(value.repository)) {
                problems.push('"repository" property must be an object')
            } else {
                if (value.repository.type && typeof value.repository.type !== 'string') {
                    problems.push('"repository" property "type" must be a string')
                }
                if (typeof value.repository.url !== 'string') {
                    problems.push('"repository" property "url" must be a string')
                }
            }
        }
        if (
            value.categories &&
            (!Array.isArray(value.categories) || !value.categories.every(category => typeof category === 'string'))
        ) {
            problems.push('"categories" property must be an array of strings')
        }
        if (value.tags && (!Array.isArray(value.tags) || !value.tags.every(tag => typeof tag === 'string'))) {
            problems.push('"tags" property must be an array of strings')
        }
        if (value.description && typeof value.description !== 'string') {
            problems.push('"description" property must be a string')
        }
        if (value.readme && typeof value.readme !== 'string') {
            problems.push('"readme" property must be a string')
        }
        if (!value.url) {
            problems.push('"url" property must be set')
        } else if (typeof value.url !== 'string') {
            problems.push('"url" property must be a string')
        }
        if (!value.activationEvents) {
            problems.push('"activationEvents" property must be set')
        } else if (!Array.isArray(value.activationEvents)) {
            problems.push('"activationEvents" property must be an array')
        } else if (!value.activationEvents.every(event => typeof event === 'string')) {
            problems.push('"activationEvents" property must be an array of strings')
        }
        if (value.contributes) {
            if (!isPlainObject(value.contributes)) {
                problems.push('"contributes" property must be an object')
            }
        }
        if (value.icon && typeof value.icon !== 'string') {
            problems.push('"icon" property must be a string')
        }
        if (problems.length > 0) {
            return new Error(`invalid extension manifest: ${problems.join(', ')}`)
        }
    }
    return value
}
