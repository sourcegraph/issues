import { applyEdits, parse as parseJSONC } from '@sqs/jsonc-parser'
import { setProperty } from '@sqs/jsonc-parser/lib/edit'
import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'
import { SettingsEdit } from '../../../../shared/src/api/client/services/settings'
import {
    mergeSettings,
    SettingsCascade,
    SettingsCascadeOrError,
    SettingsSubject,
} from '../../../../shared/src/settings/settings'
import { isErrorLike } from '../../../../shared/src/util/errors'
import { LocalStorageSubject } from '../../../../shared/src/util/LocalStorageSubject'
import { observeStorageKey, storage } from '../../browser-extension/web-extension-api/storage'
import { isInPage } from '../context'

const inPageClientSettingsKey = 'sourcegraphClientSettings'

const createStorageSettingsCascade: () => Observable<SettingsCascade> = () => {
    const storageSubject = isInPage
        ? new LocalStorageSubject<string>(inPageClientSettingsKey, '{}')
        : observeStorageKey('sync', 'clientSettings')

    const subject: SettingsSubject = {
        __typename: 'Client',
        id: 'Client',
        displayName: 'Client',
        viewerCanAdminister: true,
    }

    return storageSubject.pipe(
        map(clientSettingsString => parseJSONC(clientSettingsString || '')),
        map(clientSettings => ({
            subjects: [
                {
                    subject,
                    settings: clientSettings,
                    lastID: null,
                },
            ],
            final: clientSettings || {},
        }))
    )
}

/**
 * The settings cascade consisting solely of client settings.
 */
export const storageSettingsCascade = createStorageSettingsCascade()

/**
 * Merge two settings cascades (used to merge viewer settings and client settings).
 */
export function mergeCascades(
    cascadeOrError: SettingsCascadeOrError,
    cascade: SettingsCascade
): SettingsCascadeOrError {
    return {
        subjects:
            cascadeOrError.subjects === null
                ? cascade.subjects
                : isErrorLike(cascadeOrError.subjects)
                ? cascadeOrError.subjects
                : [...cascadeOrError.subjects, ...cascade.subjects],
        final:
            cascadeOrError.final === null
                ? cascade.final
                : isErrorLike(cascadeOrError.final)
                ? cascadeOrError.final
                : mergeSettings([cascadeOrError.final, cascade.final]),
    }
}

/**
 * Applies an edit and persists the result to client settings.
 */
export async function editClientSettings(edit: SettingsEdit | string): Promise<void> {
    const getNext = (previous: string): string =>
        typeof edit === 'string'
            ? edit
            : applyEdits(
                  previous,
                  // TODO(chris): remove `.slice()` (which guards against mutation) once
                  // https://github.com/Microsoft/node-jsonc-parser/pull/12 is merged in.
                  setProperty(previous, edit.path.slice(), edit.value, {
                      tabSize: 2,
                      insertSpaces: true,
                      eol: '\n',
                  })
              )
    if (isInPage) {
        const previous = localStorage.getItem(inPageClientSettingsKey) || ''
        const next = getNext(previous)

        localStorage.setItem(inPageClientSettingsKey, next)

        return Promise.resolve()
    }

    const { clientSettings: previous = '{}' } = await storage.sync.get()
    const next = getNext(previous)

    await storage.sync.set({ clientSettings: next })
}
