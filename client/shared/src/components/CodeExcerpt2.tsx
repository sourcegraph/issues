import * as React from 'react'
import { highlightNode } from '../util/dom'
import { toPositionOrRangeHash } from '../util/url'
import { Link } from './Link'

interface Props {
    urlWithoutPosition: string
    items: {
        line: number
        preview: string
        highlightRanges: {
            start: number
            highlightLength: number
        }[]
    }[]
    onSelect: () => void
}

export class CodeExcerpt2 extends React.PureComponent<Props> {
    public render(): JSX.Element | null {
        const maxDigits = this.props.items.reduce((digitsTotal, { line }) => {
            const digits = (line + 1).toString().length
            return digits > digitsTotal ? digits : digitsTotal
        }, 1)
        return (
            <pre className="file-match__code">
                {this.props.items.map(({ line, preview, highlightRanges }, index) => (
                    <code
                        data-line={(line + 1).toString().padStart(maxDigits)}
                        key={index}
                        ref={element =>
                            element &&
                            highlightNode(element, highlightRanges[0].start, highlightRanges[0].highlightLength)
                        }
                    >
                        <Link
                            to={`${this.props.urlWithoutPosition}${toPositionOrRangeHash({
                                position: { line: line + 1, character: highlightRanges[0].start + 1 },
                            })}`}
                            key={index}
                            onClick={this.props.onSelect}
                        >
                            {preview}
                        </Link>
                    </code>
                ))}
            </pre>
        )
    }
}
