import React, { useCallback } from 'react'
import TextareaAutosize, { TextareaAutosizeProps } from 'react-textarea-autosize'
import { Key } from 'ts-key-enum'

interface Props extends Pick<TextareaAutosizeProps, Exclude<keyof TextareaAutosizeProps, 'ref'>> {
    /**
     * Whether pressing Shift+Enter adds a newline. If false, it submits the textarea's form.
     * Pressing Enter by itself or with other modifier keys always submits the textarea's form. (If
     * you don't want Enter to ever submit the textarea's form, just use {@link TextareaAutosize}
     * directly.
     */
    newlineOnShiftEnterKeypress?: boolean
}

/**
 * A text field that starts as a single line and grows to multiple lines as needed fit its contents,
 * like the message field in Slack. Pressing Enter "submits" it; pressing Shift+Enter
 * inserts a line break.
 */
export const MultilineTextField: React.FunctionComponent<Props> = ({ newlineOnShiftEnterKeypress, ...props }) => {
    const onKeyPress: React.KeyboardEventHandler<HTMLTextAreaElement> = useCallback<
        React.KeyboardEventHandler<HTMLTextAreaElement>
    >(
        e => {
            if (e.key === Key.Enter && (!newlineOnShiftEnterKeypress || !e.shiftKey)) {
                e.preventDefault()
                if (e.currentTarget.form) {
                    e.currentTarget.form.dispatchEvent(new Event('submit'))
                }
            }
        },
        [newlineOnShiftEnterKeypress]
    )
    return <TextareaAutosize {...props} onKeyPress={onKeyPress} />
}
