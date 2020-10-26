import { from } from 'rxjs'
import { first } from 'rxjs/operators'
import { PlatformContext } from '../../../platform/context'
import { isSettingsValid } from '../../../settings/settings'

/**
 * A key path that refers to a location in a JSON document.
 *
 * Each successive array element specifies an index in an object or array to descend into. For example, in the
 * object `{"a": ["x", "y"]}`, the key path `["a", 1]` refers to the value `"y"`.
 */
export type KeyPath = (string | number)[]

/**
 * An edit to apply to settings.
 */
export interface SettingsEdit {
    /** The key path to the value. */
    path: KeyPath

    /** The new value to insert at the key path. */
    value: any
}
/**
 *
 * @todo move that to the platform context itself.
 * the code below doesn't seem to nessesery (simon)
 * @todo  also it doesn't check if subjects has any elements
 *
 */
export async function updateSettings(
    platformContext: Pick<PlatformContext, 'settings' | 'updateSettings'>,
    edit: SettingsEdit
): Promise<void> {
    const { settings: data, updateSettings: update } = platformContext
    const settings = await from(data).pipe(first()).toPromise()
    if (!isSettingsValid(settings)) {
        throw new Error('invalid settings (internal error)')
    }
    const subject = settings.subjects[settings.subjects.length - 1]
    await update(subject.subject.id, edit)
}
