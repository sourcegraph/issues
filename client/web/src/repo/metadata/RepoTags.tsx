import AddIcon from 'mdi-react/AddIcon'
import CheckIcon from 'mdi-react/CheckIcon'
import CloseIcon from 'mdi-react/CloseIcon'
import React, { useCallback, useEffect, useRef, useState } from 'react'
import { catchError } from 'rxjs/operators'

import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { IRepositoryMetadataTag } from '@sourcegraph/shared/src/graphql/schema'
import { asError, ErrorLike, isErrorLike } from '@sourcegraph/shared/src/util/errors'

import { ErrorAlert } from '../../components/alerts'
import { TreePageRepositoryFields } from '../../graphql-operations'
import { addRepoTag, deleteRepoTag, fetchRepoTags, FetchRepoTagsResult } from '../backend'

interface AddRepoTagInputProps {
    onCancel: () => void
    onSubmit: (tag: string) => void
}

const AddRepoTagInput: React.FunctionComponent<AddRepoTagInputProps> = props => {
    const reference = useRef<HTMLInputElement | null>(null)

    const submit = useCallback(() => {
        const value = reference.current?.value
        if (value === undefined) {
            props.onCancel()
        } else {
            props.onSubmit(value)
        }
    }, [props])

    const onKeyDown = useCallback(
        (event: React.KeyboardEvent<HTMLInputElement>) => {
            if (event.key === 'Escape') {
                event.stopPropagation()
                props.onCancel()
            } else if (event.key === 'Enter') {
                event.stopPropagation()
                submit()
            }
        },
        [props, submit]
    )

    const onCancel = useCallback(
        (event: React.MouseEvent) => {
            event.preventDefault()
            props.onCancel()
        },
        [props]
    )

    const onSubmit = useCallback(
        (event: React.MouseEvent) => {
            event.preventDefault()
            submit()
        },
        [submit]
    )

    return (
        <span>
            <input ref={reference} autoFocus={true} onKeyDown={onKeyDown} />
            <button className="btn btn-icon d-inline ml-1" onClick={onSubmit} type="button">
                <CheckIcon className="icon-inline" />
            </button>
            <button className="btn btn-icon d-inline ml-1" onClick={onCancel} type="button">
                <CloseIcon className="icon-inline" />
            </button>
        </span>
    )
}

interface AddRepoTagProps {
    id: string
    onUpdate?: () => void
}

const AddRepoTag: React.FunctionComponent<AddRepoTagProps> = ({ id, onUpdate }) => {
    enum State {
        Ready,
        Editable,
        Saving,
    }
    const [state, setState] = useState<State | Error>(State.Ready)

    const onAdd = useCallback(
        (event: React.MouseEvent) => {
            event.preventDefault()
            setState(State.Editable)
        },
        [State.Editable]
    )

    const onDismissError = useCallback(
        (event: React.MouseEvent) => {
            event.preventDefault()
            setState(State.Ready)
        },
        [State.Ready]
    )

    const onInputCancel = useCallback(() => {
        setState(State.Ready)
    }, [State.Ready])

    const onInputSubmit = useCallback(
        async (tag: string) => {
            if (tag.trim() === '') {
                // No point submitting a blank tag.
                setState(State.Ready)
                return
            }

            setState(State.Saving)

            try {
                await addRepoTag(id, tag)
                setState(State.Ready)
            } catch (error) {
                setState(asError(error))
            }
            onUpdate?.()
        },
        [State.Ready, State.Saving, id, onUpdate]
    )

    switch (state) {
        case State.Ready:
            return (
                <span className="badge badge-secondary">
                    <button
                        className="btn btn-icon d-inline"
                        onClick={onAdd}
                        type="button"
                        data-tooltip="Add repository tag"
                    >
                        <AddIcon className="icon-inline" />
                    </button>
                </span>
            )
        case State.Editable:
            return <AddRepoTagInput onCancel={onInputCancel} onSubmit={tag => onInputSubmit(tag)} />
        case State.Saving:
            return <LoadingSpinner className="icon-inline" />
        default:
            return (
                <>
                    <ErrorAlert error={state} />
                    <button className="btn btn-icon" onClick={onDismissError} type="button">
                        <CloseIcon className="icon-inline" />
                    </button>
                </>
            )
    }
}

type GetRepoTagsResult = FetchRepoTagsResult | ErrorLike

const getRepoTags = async (id: string): Promise<GetRepoTagsResult> =>
    fetchRepoTags({ id }, true)
        .pipe(catchError((error): [ErrorLike] => [asError(error)]))
        .toPromise()

interface RepoTagsProps {
    repo: Pick<TreePageRepositoryFields, 'id' | 'viewerCanAdminister'>
}

export const RepoTags: React.FunctionComponent<RepoTagsProps> = ({ repo: { id, viewerCanAdminister } }) => {
    // FIXME: After a good four hours on this (that is, more than the entire
    // rest of this prototype put together), I have no idea how to use RxJS to
    // wire up an observable that can be refreshed when a callback is received
    // from a child component. It seems like useEventObservable() should be able
    // to do this, but I couldn't get it to work. Instead, let's use the minimum
    // possible RxJS here and just convert it back to a promise as quickly as
    // possible so that we can use standard React from there.

    const [after, setAfter] = useState<string | undefined>(undefined)
    const [tagsOrError, setTagsOrError] = useState<GetRepoTagsResult | undefined>(undefined)

    const update = useCallback(async () => {
        setTagsOrError(await getRepoTags(id))
    }, [id, setTagsOrError])

    useEffect(() => {
        // eslint-disable-next-line @typescript-eslint/no-floating-promises
        update()
    }, [update])

    const onDeleteTag = useCallback(
        async (event: React.MouseEvent, tag: string) => {
            event.preventDefault()
            await deleteRepoTag(id, tag)
            // eslint-disable-next-line @typescript-eslint/no-floating-promises
            update()
        },
        [id, update]
    )

    const onUpdate = useCallback(() => update(), [update])

    if (tagsOrError === undefined) {
        return <LoadingSpinner className="icon-inline" />
    }
    if (isErrorLike(tagsOrError)) {
        return <ErrorAlert error={tagsOrError} prefix="Error fetching repository tags" />
    }

    return (
        <span className="ml-2">
            {tagsOrError.nodes.map(tag => (
                <span className="badge badge-secondary mr-2" key={tag.id}>
                    {tag.tag}
                    {viewerCanAdminister && (
                        <button
                            className="btn btn-icon d-inline ml-1"
                            onClick={event => onDeleteTag(event, tag.tag)}
                            type="button"
                        >
                            <CloseIcon className="icon-inline" />
                        </button>
                    )}
                </span>
            ))}
            {viewerCanAdminister && <AddRepoTag id={id} onUpdate={onUpdate} />}
        </span>
    )
}
