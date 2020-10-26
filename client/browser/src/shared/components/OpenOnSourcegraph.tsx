import * as React from 'react'
import { OpenInSourcegraphProps } from '../repo'
import { getPlatformName } from '../util/context'
import { SourcegraphIconButton, SourcegraphIconButtonProps } from './SourcegraphIconButton'
import classNames from 'classnames'

interface Props extends SourcegraphIconButtonProps {
    openProps: OpenInSourcegraphProps
}

export const OpenOnSourcegraph: React.FunctionComponent<Props> = ({
    openProps: { sourcegraphURL, repoName, revision, filePath },
    className,
    ...props
}) => {
    const url = `${sourcegraphURL}/${repoName}@${revision}/-/blob/${filePath}?utm_source=${getPlatformName()}`
    return <SourcegraphIconButton {...props} className={classNames('open-on-sourcegraph', className)} href={url} />
}
