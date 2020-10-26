import { DiffPart } from '@sourcegraph/codeintellify'
import { DOMFunctions } from '../shared/codeViews'
import { isDiffPageType, parseURL } from './util'

const getDiffCodePart = (codeElement: HTMLElement): DiffPart => {
    const tableCell = codeElement.closest('td')!

    if (tableCell.classList.contains('blob-code-addition')) {
        return 'head'
    }

    if (tableCell.classList.contains('blob-code-deletion')) {
        return 'base'
    }
    // If we can't determine the diff part the code element's parent `<td>`
    // (which may be because it is unchanged, or because the .blob-code(addition|deletion) classes
    // aren't present), call `isSplitDomDiff()`, which will look at the parent
    // code view to determine whether this is a split or unified diff view.
    if (isDomSplitDiff(codeElement)) {
        // If there are more cells on the right, this is the base, otherwise the head
        return tableCell.nextElementSibling ? 'base' : 'head'
    }

    return 'head'
}

/**
 * Returns the 0-based index of the cell that holds the line number for a given part,
 * depending on whether the diff is in unified or split view.
 * Prefers head.
 */
const getLineNumberElementIndex = (part: DiffPart, isSplitDiff: boolean): number => {
    if (part === 'base') {
        // base line number is always the first child
        return 0
    }
    return isSplitDiff ? 2 : 1
}

/**
 * Gets the line number for a given code element on unified diff, split diff and blob views
 */
const getLineNumberFromCodeElement = (codeElement: HTMLElement): number => {
    // In diff views, the code element is the `<span>` inside the cell
    // On blob views, the code element is the `<td>` itself, so `closest()` will simply return it
    // Walk all previous sibling cells until we find one with the line number
    let cell = codeElement.closest('td')!.previousElementSibling as HTMLTableCellElement
    while (cell) {
        if (cell.dataset.lineNumber) {
            return parseInt(cell.dataset.lineNumber, 10)
        }
        cell = cell.previousElementSibling as HTMLTableCellElement
    }
    throw new Error('Could not find a line number in any cell')
}

/**
 * Gets the `<td>` element for a target that contains the code
 */
const getCodeCellFromTarget = (target: HTMLElement): HTMLTableCellElement | null => {
    const cell = target.closest<HTMLTableCellElement>('td.blob-code')
    // Handle rows with the [ ↕ ] button that expands collapsed unchanged lines
    if (!cell || cell.parentElement?.classList.contains('js-expandable-line')) {
        return null
    }
    return cell
}

/**
 * Returns the `<td>` containing the code (which may contain a `.blob-code-inner`)
 */
const getDiffCodeCellFromLineNumber = (
    codeView: HTMLElement,
    line: number,
    part?: DiffPart
): HTMLTableCellElement | null => {
    if (codeView.querySelector('.js-diff-load-container')) {
        // Diff is collapsed
        return null
    }
    const isSplitDiff = isDomSplitDiff(codeView)
    const nthChild = getLineNumberElementIndex(part!, isSplitDiff) + 1 // nth-child() is 1-indexed
    const lineNumberCell = codeView.querySelector<HTMLTableCellElement>(
        `td:nth-child(${nthChild})[data-line-number="${line}"]`
    )
    if (!lineNumberCell) {
        return null
    }
    // In unified diff, the not-changed lines shall only be returned for the head.
    // Without this check they would be returned for both head and base.
    if (
        !isSplitDiff &&
        part === 'base' &&
        !lineNumberCell.classList.contains('blob-num-addition') &&
        !lineNumberCell.classList.contains('blob-num-deletion')
    ) {
        return null
    }
    let codeCell: HTMLTableCellElement
    if (isSplitDiff) {
        // In split diff view, the code cell is next to the line number cell
        codeCell = lineNumberCell.nextElementSibling as HTMLTableCellElement
    } else {
        // In unified diff view, the code cell is the last cell
        const row = lineNumberCell.parentElement as HTMLTableRowElement
        codeCell = row.lastElementChild as HTMLTableCellElement
    }
    return codeCell
}

/**
 * Returns the `<span class="blob-code-inner">` element inside a cell.
 * The code element on diff pages is the `<span class="blob-code-inner">` element inside the cell,
 * because the cell also contains a button to add a comment
 */
const getBlobCodeInner = (codeCell: HTMLTableCellElement): HTMLElement =>
    codeCell.classList.contains('blob-code-inner')
        ? codeCell // `<td>`'s in sections of the table that were expanded are not commentable so the `.blob-code-inner` element is the `<td>`
        : (codeCell.querySelector('.blob-code-inner') as HTMLElement)

/**
 * Implementations of the DOM functions for GitHub diff code views
 */
export const diffDomFunctions: DOMFunctions = {
    getCodeElementFromTarget: target => {
        const codeCell = getCodeCellFromTarget(target)
        return codeCell && getBlobCodeInner(codeCell)
    },
    getLineElementFromLineNumber: getDiffCodeCellFromLineNumber,
    getCodeElementFromLineNumber: (codeView, line, part) => {
        const codeCell = getDiffCodeCellFromLineNumber(codeView, line, part)
        return codeCell && getBlobCodeInner(codeCell)
    },
    getLineNumberFromCodeElement,
    getDiffCodePart,
    isFirstCharacterDiffIndicator: codeElement => {
        // Old versions of GitHub write a +, -, or space character directly into
        // the HTML text of the diff:
        //
        // <span class="blob-code-inner">+	fmt.<span class="pl-c1">Println</span>...
        //                               ^
        //
        // New versions of GitHub do not, and Refined GitHub used to strip these
        // characters.
        //
        // Since a +, -, or space character in the first column could be either
        //
        // - a diff indicator on an old version of GitHub, or
        // - simply part of the code being diffed on either a new version of
        //   GitHub or Refined GitHub,
        //
        // we check for the presence of other diff indicators that we know are
        // mutually exclusive with the first character diff indicator.

        // Some versions of GitHub have blob-code-marker-* classes instead of the first character diff indicator.
        const blobCodeInner = codeElement.closest('.blob-code-inner')
        const hasBlobCodeMarker =
            blobCodeInner &&
            ['deletion', 'context', 'addition'].some(name =>
                blobCodeInner.classList.contains('blob-code-marker-' + name)
            )

        // Some versions of GitHub have data-code-marker attributes instead of the first character diff indicator.
        const tableRow = codeElement.closest('tr')
        const hasDataCodeMarkerUnified = tableRow?.querySelector('td[data-code-marker]')
        const hasDataCodeMarkerSplit = blobCodeInner?.hasAttribute('data-code-marker')
        const hasDataCodeMarker = hasDataCodeMarkerUnified || hasDataCodeMarkerSplit

        // Refined GitHub used to strip the first character diff indicator.
        const hasRefinedGitHub = codeElement.closest('.refined-github-diff-signs')

        // When no other diff indicator is found, we assume the first character
        // is a diff indicator.
        return !hasBlobCodeMarker && !hasDataCodeMarker && !hasRefinedGitHub
    },
}

const getSingleFileCodeElementFromLineNumber = (codeView: HTMLElement, line: number): HTMLElement | null => {
    const lineNumberCell = codeView.querySelector(`td[data-line-number="${line}"]`)
    // In blob views, the `<td>` is the code element
    return lineNumberCell && (lineNumberCell.nextElementSibling as HTMLTableCellElement)
}

/**
 * Implementations of the DOM functions for GitHub blob code views
 */
export const singleFileDOMFunctions: DOMFunctions = {
    getCodeElementFromTarget: getCodeCellFromTarget,
    getCodeElementFromLineNumber: getSingleFileCodeElementFromLineNumber,
    getLineElementFromLineNumber: getSingleFileCodeElementFromLineNumber,
    getLineNumberFromCodeElement,
}

const getSearchCodeSnippetLineNumberCellFromLineNumber = (codeView: HTMLElement, line: number): HTMLElement | null => {
    const lineNumberCells = codeView.querySelectorAll('td.blob-num')
    let lineNumberCell: HTMLTableCellElement | null = null
    for (const cell of lineNumberCells) {
        const a = cell.querySelector('a')!
        if (a.href.endsWith(`#L${line}`)) {
            lineNumberCell = cell as HTMLTableCellElement
            break
        }
    }
    return lineNumberCell
}

const getSearchCodeSnippetCodeElementFromLineNumber = (codeView: HTMLElement, line: number): HTMLElement | null => {
    const lineNumberCell = getSearchCodeSnippetLineNumberCellFromLineNumber(codeView, line)
    // In search snippet views, the `<td>` is the code element
    return lineNumberCell && (lineNumberCell.nextElementSibling as HTMLTableCellElement)
}

export const searchCodeSnippetDOMFunctions: DOMFunctions = {
    getCodeElementFromTarget: getCodeCellFromTarget,
    getCodeElementFromLineNumber: getSearchCodeSnippetCodeElementFromLineNumber,
    getLineElementFromLineNumber: getSearchCodeSnippetCodeElementFromLineNumber,
    getLineNumberFromCodeElement: (codeElement: HTMLElement): number => {
        const cell = codeElement.closest('td')!.previousElementSibling as HTMLTableCellElement
        return parseInt(cell.firstElementChild!.textContent!, 10)
    },
}

/**
 * Returns if the current view shows diffs with split (vs. unified) view.
 *
 * @param element, either an element contained in a code view or the code view itself
 */
export function isDomSplitDiff(element: HTMLElement): boolean {
    const { pageType } = parseURL()
    if (!isDiffPageType(pageType)) {
        return false
    }
    const codeView = element.classList.contains('file') ? element : element.closest('.file')
    if (!codeView) {
        throw new Error('Could not resolve code view element')
    }
    if (codeView.classList.contains('js-comment-container')) {
        // Commented snippet in PR discussion
        return false
    }
    const codeViewTable = codeView.querySelector('table')
    if (!codeViewTable) {
        throw new Error('Could not find code view table')
    }
    return codeViewTable.classList.contains('js-file-diff-split') || codeViewTable.classList.contains('file-diff-split')
}
