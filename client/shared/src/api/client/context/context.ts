import { basename, dirname, extname } from 'path'
import { isSettingsValid, SettingsCascadeOrError } from '../../../settings/settings'
import { ViewerWithPartialModel } from '../services/viewerService'

/**
 * Context is an arbitrary, immutable set of key-value pairs. Its value can be any JSON object.
 *
 * @template T If you have a value with a property of type T that is not one of the primitive types listed below
 * (or Context), you can use Context<T> to hold that value. T must be a value that can be represented by a JSON
 * object.
 */
export interface Context<T = never>
    extends Record<
        string,
        string | number | boolean | null | Context<T> | T | (string | number | boolean | null | Context<T> | T)[]
    > {}

export type ContributionScope =
    | ViewerWithPartialModel
    | {
          type: 'panelView'
          id: string
          hasLocations: boolean
      }

/**
 * Compute the full context data based on the environment, including keys such as `resource.uri`.
 *
 * @param activeEditor the currently visible editor (if any)
 * @param settings the settings for the viewer
 * @param context manually specified context keys
 * @param scope the user interface component in whose scope this computation should occur
 */
export function computeContext<T>(
    activeEditor: ViewerWithPartialModel | undefined,
    settings: SettingsCascadeOrError,
    context: Context<T>,
    scope?: ContributionScope
): Context<T> {
    const data: Context<T> = { ...context }

    // Settings (`config.` prefix)
    if (isSettingsValid(settings)) {
        for (const [key, value] of Object.entries(settings.final)) {
            // Disable eslint warnings for any. The context is treated as a loosely typed bag of
            // values. The values for these keys are only used by extensions, whose code is in a
            // separate compilation and typechecking unit from this code. Any attempt to properly
            // reflect the types massively increases the impl complexity without any actual benefit
            // to the caller.
            //
            // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
            data[`config.${key}`] = value
        }
    }

    // Resource (`resource.` prefix)
    const component: ContributionScope | null = scope || activeEditor || null
    if (component) {
        data.component = true
    }
    if (component?.type === 'CodeEditor') {
        data.resource = true

        // TODO(sqs): Define these precisely. If the resource is in a repository, what is the "path"? Is it the
        // path relative to the repository's root? If it's a file on disk, then "path" could also mean the
        // (absolute) path on the file system. Clear up that ambiguity.
        data['resource.uri'] = component.resource
        data['resource.basename'] = basename(component.resource)
        data['resource.dirname'] = dirname(component.resource)
        data['resource.extname'] = extname(component.resource)
        data['resource.language'] = component.model.languageId
        data['resource.type'] = 'textDocument'

        data['component.type'] = 'CodeEditor'
        // See above for why we disable eslint rules related to `any`.
        //
        // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
        data['component.selections'] = component.selections as any // eslint-disable-line @typescript-eslint/no-explicit-any
        if (component.selections.length > 0) {
            // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
            data['component.selection'] = (component.selections[0] || null) as any // eslint-disable-line @typescript-eslint/no-explicit-any
            // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
            data['component.selection.start'] = (component.selections[0] ? component.selections[0].start : null) as any // eslint-disable-line @typescript-eslint/no-explicit-any
            // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
            data['component.selection.end'] = (component.selections[0] ? component.selections[0].end : null) as any // eslint-disable-line @typescript-eslint/no-explicit-any
            data['component.selection.start.line'] = component.selections[0] ? component.selections[0].start.line : null
            data['component.selection.start.character'] = component.selections[0]
                ? component.selections[0].start.character
                : null
            data['component.selection.end.line'] = component.selections[0] ? component.selections[0].end.line : null
            data['component.selection.end.character'] = component.selections[0]
                ? component.selections[0].end.character
                : null
        }
    }

    // Panel (`panel.` prefix)
    if (component?.type === 'panelView') {
        data['panel.activeView.id'] = component.id
        data['panel.activeView.hasLocations'] = component.hasLocations
    }

    return data
}
