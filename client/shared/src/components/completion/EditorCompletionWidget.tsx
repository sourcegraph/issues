import { Position, Range } from '@sourcegraph/extension-api-types'
import { isEqual } from 'lodash'
import React, { useEffect, useState, useCallback } from 'react'
import { merge, of } from 'rxjs'
import {
    catchError,
    debounceTime,
    delay,
    distinctUntilChanged,
    first,
    share,
    switchMap,
    takeUntil,
    throttleTime,
} from 'rxjs/operators'
import { CompletionItem, CompletionList } from 'sourcegraph'
import { offsetToPosition, positionToOffset } from '../../api/client/types/textDocument'
import { ExtensionsControllerProps } from '../../extensions/controller'
import { asError, ErrorLike } from '../../util/errors'
import { throttleTimeWindow } from '../../util/rxjs/throttleTimeWindow'
import { getWordAtText } from '../../util/wordHelpers'
import { CompletionWidget, CompletionWidgetProps } from './CompletionWidget'
import { observeEditorAndModel } from '../../api/client/services/viewerService'

export interface EditorCompletionWidgetProps
    extends ExtensionsControllerProps,
        Pick<CompletionWidgetProps, Exclude<keyof CompletionWidgetProps, 'completionListOrError' | 'onSelectItem'>> {
    /**
     * The ID of the editor to show a completion widget for.
     */
    viewerId: string
}

const LOADING = 'loading' as const

/**
 * Shows a completion widget with a list of completion items from extensions for a given editor.
 */
export const EditorCompletionWidget: React.FunctionComponent<EditorCompletionWidgetProps> = ({
    extensionsController: {
        services: { viewer: viewerService, model: modelService, completionItems: completionItemsService },
    },
    viewerId,
    textArea,
    ...props
}) => {
    const [completionListOrError, setCompletionListOrError] = useState<
        typeof LOADING | CompletionList | null | ErrorLike
    >(null)
    useEffect(() => {
        const subscription = observeEditorAndModel({ viewerId }, viewerService, modelService)
            .pipe(
                debounceTime(0), // Debounce multiple synchronous changes so we only handle them once.
                // These throttles are tweaked for maximum perceived responsiveness. They can
                // probably be made even more responsive (more lenient throttling) when
                // https://github.com/sourcegraph/sourcegraph/issues/3433 is fixed.
                //
                // It is OK to drop intermediate events because the events themselves aren't used,
                // only the resulting state. And throttleTimeWindow always emits the trailing event,
                // so we never skip an update.
                throttleTime(100, undefined, { leading: true, trailing: true }),
                throttleTimeWindow(500, 2),
                distinctUntilChanged((a, b) => isEqual(a.selections, b.selections) && a.model.text === b.model.text),
                switchMap(editor => {
                    if (editor.selections.length === 0) {
                        return of(null)
                    }
                    const result = completionItemsService
                        .getCompletionItems({
                            textDocument: { uri: editor.resource },
                            position: editor.selections[0].active,
                        })
                        .pipe(share())
                    return merge(of(LOADING).pipe(delay(2000), takeUntil(result)), result)
                }),
                catchError(error => [asError(error)])
            )
            .subscribe(setCompletionListOrError)
        return () => subscription.unsubscribe()
    }, [completionItemsService, viewerId, viewerService, viewerService.viewers, modelService])

    const onSelectItem = useCallback(
        async (item: CompletionItem): Promise<void> => {
            const editor = await observeEditorAndModel({ viewerId }, viewerService, modelService)
                .pipe(first())
                .toPromise()
            const [sel, ...secondarySelections] = editor.selections
            if (!sel) {
                throw new Error('no selection')
            }
            if (!editor.model.text) {
                throw new Error('model text not available')
            }

            let replaceRange: Range
            const word = getWordAtText(positionToOffset(editor.model.text, sel.active), editor.model.text)
            if (word) {
                replaceRange = {
                    start: offsetToPosition(editor.model.text, word.startColumn),
                    end: offsetToPosition(editor.model.text, word.endColumn),
                }
            } else {
                replaceRange = sel
            }

            const beforeText = editor.model.text.slice(0, positionToOffset(editor.model.text, replaceRange.start))
            const afterText = editor.model.text.slice(positionToOffset(editor.model.text, replaceRange.end))
            const itemText = item.insertText !== undefined ? item.insertText : item.label
            modelService.updateModel(editor.resource, beforeText + itemText + afterText)

            // TODO: Support multi-line completion insertions.
            const position: Position = {
                line: replaceRange.start.line,
                character: replaceRange.start.character + itemText.length,
            }
            viewerService.setSelections(editor, [
                {
                    active: position,
                    anchor: position,
                    start: position,
                    end: position,
                    isReversed: false,
                },
                ...secondarySelections,
            ])

            setCompletionListOrError(null)
        },
        [modelService, viewerId, viewerService]
    )

    return (
        <CompletionWidget
            {...props}
            completionListOrError={completionListOrError}
            textArea={textArea}
            onSelectItem={onSelectItem}
        />
    )
}
