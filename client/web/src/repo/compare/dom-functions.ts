import { DiffPart, DOMFunctions } from '@sourcegraph/codeintellify'

export const diffDomFunctions: DOMFunctions = {
    getCodeElementFromTarget: (target: HTMLElement): HTMLTableCellElement | null => {
        const row = target.closest('tr')
        if (!row) {
            return null
        }
        return row.cells[2]
    },

    getCodeElementFromLineNumber: (
        codeView: HTMLElement,
        line: number,
        part?: DiffPart
    ): HTMLTableCellElement | null => {
        // For unchanged lines, prefer line number in head
        const lineNumberCell = codeView.querySelector(`[data-line="${line}"][data-part="${part || 'head'}"]`)
        if (!lineNumberCell) {
            return null
        }
        const row = lineNumberCell.parentElement as HTMLTableRowElement
        const codeCell = row.cells[2]
        return codeCell
    },

    getLineNumberFromCodeElement: (codeCell: HTMLElement): number => {
        const row = codeCell.closest('tr')
        if (!row) {
            throw new Error('Could not find closest row for codeCell')
        }
        const [baseLineNumberCell, headLineNumberCell] = row.cells
        // For unchanged lines, prefer line number in head
        if (headLineNumberCell.dataset.line) {
            return +headLineNumberCell.dataset.line
        }
        if (baseLineNumberCell.dataset.line) {
            return +baseLineNumberCell.dataset.line
        }
        throw new Error('Neither head or base line number cell have data-line set')
    },

    getDiffCodePart: (codeCell: HTMLElement): DiffPart => {
        const row = codeCell.parentElement as HTMLTableRowElement
        const [baseLineNumberCell, headLineNumberCell] = row.cells
        if (baseLineNumberCell.dataset.part && headLineNumberCell.dataset.part) {
            return null
        }
        if (baseLineNumberCell.dataset.part) {
            return 'base'
        }
        if (headLineNumberCell.dataset.part) {
            return 'head'
        }
        throw new Error('Could not figure out diff part for code element')
    },

    isFirstCharacterDiffIndicator: () => false,
}
