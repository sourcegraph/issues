import { DiffPart } from '@sourcegraph/codeintellify'
import { DOMFunctions } from '../shared/codeViews'

/**
 * Returns `true` if the element is a line number cell in a Phabricator diff code views.
 *
 * Supports both `<th>` line number cells, where the line number is the `textContent` (old Phabricator versions)
 * and `<td>` line number cells with a `data-n` attribtue (recent Phabricator versions).
 */
const isLineNumberCell = (element: HTMLElement): boolean =>
    Boolean((element.tagName === 'TH' && element.textContent) || element.dataset.n)

const getLineNumber = (lineNumberCell: HTMLElement): number =>
    parseInt((lineNumberCell.tagName === 'TH' ? lineNumberCell.textContent : lineNumberCell.dataset.n) || '', 10)

/**
 * Returns the closest line number cell to a code element in a Phabricator diff.
 * If no line number cell can be found, an error is thrown.
 *
 * Supports both `<th>` line number cells, where the line number is the `textContent` (old Phabricator versions)
 * and `<td>` line number cells with a `data-n` attribtue (recent Phabricator versions).
 */
const getLineNumberCellFromCodeElement = (codeElement: HTMLElement): HTMLElement | null => {
    let element: HTMLElement | null = codeElement
    while (element) {
        if (isLineNumberCell(element)) {
            return element
        }
        element = element.previousElementSibling as HTMLElement | null
    }
    return null
}

const getDiffLineNumberElementFromLineNumber = (
    codeView: HTMLElement,
    line: number,
    part?: DiffPart
): HTMLElement | null => {
    const lineNumberSelector = codeView.querySelector('td[data-n]')
        ? 'td[data-n]'
        : `th:nth-of-type(${part === 'base' ? 1 : 2})`
    for (const lineNumberCell of codeView.querySelectorAll<HTMLElement>(lineNumberSelector)) {
        if (getLineNumber(lineNumberCell) === line) {
            if (part === 'head' && lineNumberCell.previousElementSibling === null) {
                // this is the line number for the base element
                continue
            }
            return lineNumberCell
        }
    }
    return null
}

const getDiffCodeElementFromLineNumber = (codeView: HTMLElement, line: number, part?: DiffPart): HTMLElement | null => {
    const lineNumberCell = getDiffLineNumberElementFromLineNumber(codeView, line, part)
    let codeElement: HTMLElement | null = lineNumberCell
    while (
        codeElement &&
        // On unified diffs, some <th> or td.n elements can have an empty text content or no data-n attribute
        // (for added lines that did not exist in the base, for instance),
        // in which case isLineNumberCell returns false.
        (codeElement.tagName !== 'TD' ||
            codeElement.classList.contains('n') ||
            isLineNumberCell(codeElement) ||
            codeElement.classList.contains('copy'))
    ) {
        codeElement = codeElement.nextElementSibling as HTMLElement | null
    }
    return codeElement
}

/**
 * Implementations of the DOM functions for diff code views on Phabricator
 */
export const diffDomFunctions: DOMFunctions = {
    getCodeElementFromTarget: target => {
        if (target.tagName === 'TH' || target.classList.contains('copy')) {
            return null
        }

        const tableCell = target.closest('td')
        if (!tableCell) {
            return null
        }
        if (tableCell.classList.contains('show-more') || tableCell.classList.contains('show-context')) {
            // This element represents a collapsed part of the diff, it's not a code element.
            return null
        }
        if (!getLineNumberCellFromCodeElement(tableCell)) {
            // The element has no associated line number cell: this can be the case when hovering
            // 'empty' lines in the base part of a split diff that has added lines.
            return null
        }
        return tableCell
    },
    getCodeElementFromLineNumber: getDiffCodeElementFromLineNumber,
    getLineElementFromLineNumber: getDiffCodeElementFromLineNumber,
    getLineNumberFromCodeElement: codeElement => {
        const lineNumberCell = getLineNumberCellFromCodeElement(codeElement)
        if (!lineNumberCell) {
            throw new Error('Could not find line number cell from code element')
        }
        return getLineNumber(lineNumberCell)
    },
    getDiffCodePart: codeElement => {
        // Changed lines have handy class names.
        if (codeElement.classList.contains('old')) {
            return 'base'
        }
        if (codeElement.classList.contains('new')) {
            return 'head'
        }

        const lineNumberCell = getLineNumberCellFromCodeElement(codeElement)
        if (!lineNumberCell) {
            throw new Error('Could not find line number cell from code element')
        }

        // In unified diffs, both <th>'s have a class telling us which side of the diff the line belongs to.
        if (lineNumberCell.classList.contains('left')) {
            return 'base'
        }
        if (lineNumberCell.classList.contains('right')) {
            return 'head'
        }

        // If the lineNumberCell is the first element in the line, the codeElement
        // belongs to the base part of the diff.
        return lineNumberCell.previousElementSibling ? 'head' : 'base'
    },
    isFirstCharacterDiffIndicator: (codeElement: HTMLElement) => {
        const firstChild = codeElement.firstElementChild as HTMLElement
        if (firstChild?.classList.contains('aural-only')) {
            return true
        }

        return false
    },
}

const getDiffusionCodeElementFromLineNumber = (
    codeView: HTMLElement,
    line: number,
    part?: DiffPart
): HTMLElement | null => {
    const row = codeView.querySelector<HTMLTableRowElement>(`tr:nth-of-type(${line})`)
    if (!row) {
        throw new Error(`unable to find row ${line} from code view`)
    }
    return row.querySelector('td')
}

export const diffusionDOMFns: DOMFunctions = {
    getCodeElementFromTarget: target => target.closest('td'),
    getCodeElementFromLineNumber: getDiffusionCodeElementFromLineNumber,
    getLineElementFromLineNumber: getDiffusionCodeElementFromLineNumber,
    getLineNumberFromCodeElement: codeElement => {
        let lineCell = codeElement as HTMLElement | null
        while (
            lineCell !== null &&
            lineCell.tagName !== 'TH' &&
            !lineCell.classList.contains('phabricator-source-line')
        ) {
            lineCell = lineCell.previousElementSibling as HTMLElement | null
        }
        if (!lineCell) {
            throw new Error('could not find line number cell from code element')
        }

        const lineAnchor = lineCell.querySelector('a')
        if (!lineAnchor) {
            throw new Error('could not find line number anchor from code element')
        }
        // In recent Phabricator versions, the line number is stored in the `data-n`
        // attribute, and the textContent is empty.
        if (lineAnchor.dataset.n !== undefined) {
            const lineNumber = parseInt(lineAnchor.dataset.n, 10)
            if (isNaN(lineNumber)) {
                throw new TypeError(`Could not parse lineNumber from data-n attribute: ${lineAnchor.dataset.n}`)
            }
            return lineNumber
        }
        const lineNumber = parseInt(lineAnchor.textContent || '', 10)
        if (isNaN(lineNumber)) {
            throw new TypeError(
                `Could not parse lineNumber from lineAnchor.textContent: ${String(lineAnchor.textContent)}`
            )
        }
        return lineNumber
    },
    isFirstCharacterDiffIndicator: () => false,
}
