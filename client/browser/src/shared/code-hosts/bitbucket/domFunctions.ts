import { DiffPart } from '@sourcegraph/codeintellify'
import { DOMFunctions } from '../shared/codeViews'

const getSingleFileLineElementFromLineNumber = (codeView: HTMLElement, line: number): HTMLElement => {
    const lineNumberElement = codeView.querySelector<HTMLElement>(`[data-line-number="${line}"]`)
    if (!lineNumberElement) {
        throw new Error(`Line ${line} not found in code view`)
    }

    const lineElement = lineNumberElement.closest<HTMLElement>('.line')
    if (!lineElement) {
        throw new Error('Could not find line elem for line element')
    }

    return lineElement
}

export const singleFileDOMFunctions: DOMFunctions = {
    getCodeElementFromTarget: target => {
        const container = target.closest('.CodeMirror-line')

        return container ? container.querySelector<HTMLElement>('span[role="presentation"]') : null
    },
    getLineNumberFromCodeElement: codeElement => {
        const line = codeElement.closest('.line')
        if (!line) {
            throw new Error('Could not find line containing code element')
        }

        const lineNumberElement = line.querySelector<HTMLElement>('.line-locator')
        if (!lineNumberElement) {
            throw new Error('Could not find the line number in a line container')
        }

        const lineNumber = parseInt(lineNumberElement.dataset.lineNumber || '', 10)
        if (isNaN(lineNumber)) {
            throw new TypeError('data-line-number not set on line number element')
        }

        return lineNumber
    },
    getLineElementFromLineNumber: getSingleFileLineElementFromLineNumber,
    getCodeElementFromLineNumber: (codeView, line) =>
        getSingleFileLineElementFromLineNumber(codeView, line).querySelector<HTMLElement>(
            '.CodeMirror-line span[role="presentation"]'
        ),
}

const getDiffLineElementFromLineNumber = (codeView: HTMLElement, line: number, part?: DiffPart): HTMLElement => {
    for (const lineNumberElement of codeView.querySelectorAll(`.line-number-${part === 'head' ? 'to' : 'from'}`)) {
        const lineNumber = parseInt((lineNumberElement.textContent || '').trim(), 10)
        if (!isNaN(lineNumber) && lineNumber === line) {
            const lineElement = lineNumberElement.closest<HTMLElement>('.line')
            if (!lineElement) {
                throw new Error('Could not find lineElem from lineNumElem')
            }

            return lineElement
        }
    }

    throw new Error(`Could not locate line number element for line ${line}, part: ${String(part)}`)
}

export const diffDOMFunctions: DOMFunctions = {
    getCodeElementFromTarget: singleFileDOMFunctions.getCodeElementFromTarget,
    getLineNumberFromCodeElement: codeElement => {
        const line = codeElement.closest('.line')
        if (!line) {
            throw new Error('Could not find line containing code element')
        }

        const lineNumberTo = line.querySelector<HTMLElement>('.line-number-to')
        if (lineNumberTo) {
            const lineNumber = parseInt((lineNumberTo.textContent || '').trim(), 10)
            if (!isNaN(lineNumber)) {
                return lineNumber
            }
        }

        const lineNumberFrom = line.querySelector<HTMLElement>('.line-number-from')
        if (lineNumberFrom) {
            const lineNumber = parseInt((lineNumberFrom.textContent || '').trim(), 10)
            if (!isNaN(lineNumber)) {
                return lineNumber
            }
        }

        throw new Error('Could not find line number element for code element')
    },
    getLineElementFromLineNumber: getDiffLineElementFromLineNumber,
    getCodeElementFromLineNumber: (codeView, line, part) =>
        getDiffLineElementFromLineNumber(codeView, line, part).querySelector<HTMLElement>(
            '.CodeMirror-line span[role="presentation"]'
        ),
    getDiffCodePart: codeElement => {
        if (!document.querySelector('.side-by-side-diff')) {
            return codeElement.closest('.line')!.classList.contains('removed') ? 'base' : 'head'
        }

        const diffSide = codeElement.closest('.diff-editor')!

        // If the sibling to the left is the diff divider, it's in the HEAD.
        return diffSide.previousElementSibling?.classList.contains('segment-connector-column') ? 'head' : 'base'
    },
    isFirstCharacterDiffIndicator: () => false,
}
