import classNames from 'classnames'
import InfoCircleOutlineIcon from 'mdi-react/InfoCircleOutlineIcon'
import React, { Fragment, useCallback, useMemo, useState } from 'react'

import { ChangesetState } from '@sourcegraph/shared/src/graphql-operations'
import { pluralize } from '@sourcegraph/shared/src/util/strings'

import { AllChangesetIDsVariables, Scalars } from '../../../../graphql-operations'
import { eventLogger } from '../../../../tracking/eventLogger'
import { queryAllChangesetIDs } from '../backend'

import styles from './ChangesetSelectRow.module.scss'
import { CloseChangesetsModal } from './CloseChangesetsModal'
import { CreateCommentModal } from './CreateCommentModal'
import { DetachChangesetsModal } from './DetachChangesetsModal'
import { MergeChangesetsModal } from './MergeChangesetsModal'
import { PublishChangesetsModal } from './PublishChangesetsModal'
import { ReenqueueChangesetsModal } from './ReenqueueChangesetsModal'

/**
 * Describes a possible action on the changeset list.
 */
interface ChangesetListAction {
    /* The type of action. Used internally. */
    type: string
    /* The button label for the action. */
    buttonLabel: string
    /* The title in the dropdown menu item. */
    dropdownTitle: string
    /* The description in the dropdown menu item. */
    dropdownDescription: string
    /* Conditionally display the action based on the given query arguments. */
    isAvailable: (queryArguments: Omit<AllChangesetIDsVariables, 'after'>) => boolean
    /**
     * Invoked when the action is triggered. Either onDone or onCancel need to be called
     * eventually. Can return a JSX.Element to be rendered adacent to the button (i.e. a modal).
     */
    onTrigger: (
        batchChangeID: Scalars['ID'],
        changesetIDs: () => Promise<Scalars['ID'][]>,
        onDone: () => void,
        onCancel: () => void
    ) => void | JSX.Element
    experimental?: boolean
}

const AVAILABLE_ACTIONS: ChangesetListAction[] = [
    {
        type: 'detach',
        buttonLabel: 'Detach changesets',
        dropdownTitle: 'Detach changesets',
        dropdownDescription:
            "Remove the selected changesets from this batch change. Unlike archive, this can't be undone.",
        // Only show on the archived tab.
        isAvailable: ({ onlyArchived }) => !!onlyArchived,
        onTrigger: (batchChangeID, changesetIDs, onDone, onCancel) => (
            <DetachChangesetsModal
                batchChangeID={batchChangeID}
                changesetIDs={changesetIDs}
                afterCreate={onDone}
                onCancel={onCancel}
                telemetryService={eventLogger}
            />
        ),
    },
    {
        type: 'retry',
        buttonLabel: 'Retry changesets',
        dropdownTitle: 'Retry changesets',
        dropdownDescription: 'Re-enqueues the selected changesets for processing, if they failed.',
        // Only show when filtering by state === FAILED:
        isAvailable: ({ state }) => state === ChangesetState.FAILED,
        onTrigger: (batchChangeID, changesetIDs, onDone, onCancel) => (
            <ReenqueueChangesetsModal
                batchChangeID={batchChangeID}
                changesetIDs={changesetIDs}
                afterCreate={onDone}
                onCancel={onCancel}
            />
        ),
    },
    {
        type: 'commentatore',
        buttonLabel: 'Create comment',
        dropdownTitle: 'Create comment',
        dropdownDescription:
            'Create a comment on all selected changesets. For example, you could ask people for reviews, give an update, or post a cat GIF.',
        isAvailable: () => true,
        onTrigger: (batchChangeID, changesetIDs, onDone, onCancel) => (
            <CreateCommentModal
                batchChangeID={batchChangeID}
                changesetIDs={changesetIDs}
                afterCreate={onDone}
                onCancel={onCancel}
            />
        ),
    },
    {
        type: 'merge',
        experimental: true,
        buttonLabel: 'Merge changesets',
        dropdownTitle: 'Merge changesets',
        dropdownDescription:
            'Attempt to merge all selected changesets. Some changesets may be unmergeable if there are rules preventing merge, such as CI requirements.',
        isAvailable: ({ state }) => state === ChangesetState.OPEN,
        onTrigger: (batchChangeID, changesetIDs, onDone, onCancel) => (
            <MergeChangesetsModal
                batchChangeID={batchChangeID}
                changesetIDs={changesetIDs}
                afterCreate={onDone}
                onCancel={onCancel}
            />
        ),
    },
    {
        type: 'close',
        buttonLabel: 'Close changesets',
        dropdownTitle: 'Close changesets',
        dropdownDescription:
            'Attempt to close all selected changesets on the code hosts. The changesets will remain part of the batch change.',
        isAvailable: ({ state }) => state === ChangesetState.OPEN || state === ChangesetState.DRAFT,
        onTrigger: (batchChangeID, changesetIDs, onDone, onCancel) => (
            <CloseChangesetsModal
                batchChangeID={batchChangeID}
                changesetIDs={changesetIDs}
                afterCreate={onDone}
                onCancel={onCancel}
            />
        ),
    },
    {
        type: 'publish',
        buttonLabel: 'Publish changesets',
        dropdownTitle: 'Publish changesets',
        dropdownDescription: 'Attempt to publish all selected changesets to the code hosts.',
        isAvailable: ({ state }) => state !== ChangesetState.CLOSED,
        onTrigger: (batchChangeID, changesetIDs, onDone, onCancel) => (
            <PublishChangesetsModal
                batchChangeID={batchChangeID}
                changesetIDs={changesetIDs}
                afterCreate={onDone}
                onCancel={onCancel}
            />
        ),
    },
]

export interface ChangesetSelectRowProps {
    selected: Set<Scalars['ID']>
    batchChangeID: Scalars['ID']
    onSubmit: () => void
    allVisibleSelected: boolean
    totalCount: number
    allSelected: boolean
    setAllSelected: () => void
    queryArguments: Omit<AllChangesetIDsVariables, 'after'>

    /** For testing only. */
    dropDownInitiallyOpen?: boolean
}

/**
 * Renders the top bar of the ChangesetList with the action buttons and the X selected
 * label. Provides select ALL functionality.
 */
export const ChangesetSelectRow: React.FunctionComponent<ChangesetSelectRowProps> = ({
    selected,
    batchChangeID,
    onSubmit,
    allVisibleSelected,
    totalCount,
    allSelected,
    setAllSelected,
    queryArguments,
    dropDownInitiallyOpen = false,
}) => {
    const actions = useMemo(() => AVAILABLE_ACTIONS.filter(action => action.isAvailable(queryArguments)), [
        queryArguments,
    ])
    /* Whether the dropdown menu is expanded. */
    const [isOpen, setIsOpen] = useState<boolean>(dropDownInitiallyOpen)
    /* Toggle the dropdown menu */
    const toggleIsOpen = useCallback(() => setIsOpen(open => !open), [])
    const [selectedAction, setSelectedAction] = useState<ChangesetListAction | undefined>(() => {
        // If there's only one available action, default select that one.
        if (actions.length === 1) {
            return actions[0]
        }
        return undefined
    })
    const onSelectedTypeSelect = useCallback(
        (type: string) => {
            setSelectedAction(actions.find(action => action.type === type))
            setIsOpen(false)
        },
        [actions]
    )
    const [renderedElement, setRenderedElement] = useState<JSX.Element | undefined>()
    const onTriggerAction = useCallback(() => {
        if (!selectedAction) {
            return
        }
        // Depending on the selection, we need to construct a loader function for
        // the changeset IDs.
        let ids: () => Promise<Scalars['ID'][]>
        if (allSelected) {
            // We asynchronously fetch all the IDs for ALL all.
            ids = () => queryAllChangesetIDs(queryArguments).toPromise()
        } else {
            // We can just pass down the IDs.
            ids = () => Promise.resolve([...selected])
        }
        const element = selectedAction.onTrigger(
            batchChangeID,
            ids,
            onSubmit,
            // On cancel hide the rendered element.
            () => {
                setRenderedElement(undefined)
            }
        )
        if (element !== undefined) {
            setRenderedElement(element)
        }
    }, [allSelected, batchChangeID, onSubmit, queryArguments, selected, selectedAction])

    let buttonLabel = selectedAction === undefined ? 'Select action' : selectedAction.buttonLabel
    if (selectedAction?.experimental) {
        buttonLabel += ' (Experimental)'
    }

    // If we have ALL all selected, we take the totalCount in the current connection, otherwise the count of selected changeset IDs.
    const selectedAmount = allSelected ? totalCount : selected.size

    return (
        <>
            {renderedElement}
            <div className="row align-items-center no-gutters">
                <div className="ml-2 col d-flex align-items-center">
                    <InfoCircleOutlineIcon className="icon-inline text-muted mr-2" />
                    {selectedAmount} {pluralize('changeset', selectedAmount)} selected
                    {allVisibleSelected && totalCount > selectedAmount && (
                        <button type="button" className="btn btn-link py-0 px-1" onClick={setAllSelected}>
                            (Select all {totalCount})
                        </button>
                    )}
                </div>
                <div className="w-100 d-block d-md-none" />
                <div className="m-0 col col-md-auto">
                    <div className="row no-gutters">
                        <div className="col my-2 ml-0 ml-sm-2">
                            <div className="btn-group">
                                <button
                                    type="button"
                                    className="btn btn-primary text-nowrap"
                                    onClick={onTriggerAction}
                                    disabled={selected.size === 0 || selectedAction === undefined}
                                >
                                    {buttonLabel}
                                </button>
                                {actions.length > 1 && (
                                    <>
                                        <button
                                            type="button"
                                            onClick={toggleIsOpen}
                                            className="btn btn-primary dropdown-toggle dropdown-toggle-split"
                                        />
                                        <div
                                            className={classNames(
                                                styles.changesetSelectRowDropdownItem,
                                                'dropdown-menu dropdown-menu-right',
                                                isOpen && 'show'
                                            )}
                                        >
                                            {actions.map((action, index) => (
                                                <Fragment key={action.type}>
                                                    <ActionDropdownItem
                                                        action={action}
                                                        setSelectedType={onSelectedTypeSelect}
                                                    />
                                                    {index !== actions.length - 1 && (
                                                        <div className="dropdown-divider" />
                                                    )}
                                                </Fragment>
                                            ))}
                                        </div>
                                    </>
                                )}
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </>
    )
}

interface ActionDropdownItemProps {
    setSelectedType: (type: string) => void
    action: ChangesetListAction
}

const ActionDropdownItem: React.FunctionComponent<ActionDropdownItemProps> = ({ action, setSelectedType }) => {
    const onClick = useCallback<React.MouseEventHandler>(() => {
        setSelectedType(action.type)
    }, [setSelectedType, action.type])
    return (
        <div className="dropdown-item">
            <button type="button" className="btn text-left" onClick={onClick}>
                <h4 className="mb-1">
                    {action.dropdownTitle}
                    {action.experimental && (
                        <>
                            {' '}
                            <small className="badge badge-info">Experimental</small>
                        </>
                    )}
                </h4>
                <p className="text-wrap text-muted mb-0">
                    <small>{action.dropdownDescription}</small>
                </p>
            </button>
        </div>
    )
}
