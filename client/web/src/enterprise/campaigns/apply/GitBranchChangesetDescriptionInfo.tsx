import React, { useCallback, useState } from 'react'
import { GitBranchChangesetDescriptionFields } from '../../../graphql-operations'
import { formatPersonName, PersonLink } from '../../../person/PersonLink'
import { UserAvatar } from '../../../user/UserAvatar'

const dotDotDot = '\u22EF'

interface Props {
    description: GitBranchChangesetDescriptionFields

    /** For testing only. */
    isExpandedInitially?: boolean
}

export const GitBranchChangesetDescriptionInfo: React.FunctionComponent<Props> = ({
    description,
    isExpandedInitially = false,
}) => {
    const [showCommitMessageBody, setShowCommitMessageBody] = useState<boolean>(isExpandedInitially)

    const toggleShowCommitMessageBody = useCallback((): void => {
        setShowCommitMessageBody(!showCommitMessageBody)
    }, [showCommitMessageBody])

    return (
        <>
            {description.commits.map((commit, index) => (
                <React.Fragment key={index}>
                    <div className="w-100">
                        <div className="d-flex align-items-center flex-grow-1">
                            <small className="mr-2">
                                <UserAvatar
                                    className="icon-inline mr-1"
                                    user={commit.author}
                                    data-tooltip={formatPersonName(commit.author)}
                                />{' '}
                                <PersonLink person={commit.author} className="font-weight-bold" />
                            </small>
                            <span
                                className="overflow-hidden font-weight-bold text-nowrap text-truncate pr-2"
                                data-tooltip={commit.subject}
                            >
                                {commit.subject}
                            </span>
                            {commit.body && (
                                <button
                                    type="button"
                                    className="btn btn-secondary btn-sm px-1 py-0 font-weight-bold align-item-center mr-2"
                                    onClick={toggleShowCommitMessageBody}
                                >
                                    {dotDotDot}
                                </button>
                            )}
                        </div>
                    </div>
                    {showCommitMessageBody && (
                        <div className="w-100 mt-1">
                            <small>
                                <pre className="text-wrap mb-0">{commit.body}</pre>
                            </small>
                        </div>
                    )}
                    <hr className="mb-3" />
                </React.Fragment>
            ))}
        </>
    )
}
