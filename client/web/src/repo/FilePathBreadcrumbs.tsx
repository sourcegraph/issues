import * as React from 'react'
import { LinkOrSpan } from '../../../shared/src/components/LinkOrSpan'
import { RepoRevision, toPrettyBlobURL } from '../../../shared/src/util/url'
import { toTreeURL } from '../util/url'
import classNames from 'classnames'

/**
 * Displays a file path in a repository in breadcrumb style, with ancestor path
 * links.
 */
export const FilePathBreadcrumbs: React.FunctionComponent<
    RepoRevision & {
        filePath: string
        isDir: boolean
    }
> = ({ repoName, revision, filePath, isDir }) => {
    const parts = filePath.split('/')
    const partToUrl = (index: number): string => {
        const partPath = parts.slice(0, index + 1).join('/')
        if (isDir || index < parts.length - 1) {
            return toTreeURL({ repoName, revision, filePath: partPath })
        }
        return toPrettyBlobURL({ repoName, revision, filePath: partPath })
    }
    const partToClassName = (index: number): string =>
        index === parts.length - 1
            ? 'part-last test-breadcrumb-part-last'
            : 'part-directory test-breadcrumb-part-directory'

    const spans: JSX.Element[] = []
    for (const [index, part] of parts.entries()) {
        const link = partToUrl(index)
        const className = classNames('part', partToClassName?.(index))
        spans.push(
            <LinkOrSpan
                key={index}
                className={className}
                to={link}
                aria-current={index === parts.length - 1 ? 'page' : 'false'}
            >
                {part}
            </LinkOrSpan>
        )
        if (index < parts.length - 1) {
            spans.push(
                <span key={`sep${index}`} className="file-path-breadcrumbs__separator text-muted font-weight-semibold">
                    /
                </span>
            )
        }
    }
    return (
        // Important: do not put spaces between the breadcrumbs or spaces will get added when copying the path
        <span className="file-path-breadcrumbs">{spans}</span>
    )
}
