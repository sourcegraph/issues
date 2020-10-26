import * as dom from './dom'

describe('util/dom', () => {
    describe('highlightNode', () => {
        const cellInnerHTML =
            '<span style="color:#c0c5ce;"><span>\t</span></span><span style="color:#fff3bf;"><span>ServeHTTP</span></span><span style="color:#c0c5ce;"><span>(</span></span><span style="color:#c0c5ce;"><span>ResponseWriter</span></span><span style="color:#c0c5ce;"><span>,</span></span><span style="color:#c0c5ce;"><span> </span></span><span style="color:#329af0;"><span>*</span></span><span style="color:#c0c5ce;"><span>Request</span></span><span style="color:#c0c5ce;"><span>)</span></span>'
        let cell: HTMLTableCellElement

        beforeEach(() => {
            document.body.innerHTML = `<table><tbody><td id="cell">${cellInnerHTML}</td></tbody></table>`
            cell = window.document.querySelector<HTMLTableCellElement>('#cell')!
        })

        test('highlights no characters', () => {
            dom.highlightNode(cell, 0, 0)
            expect(cell.innerHTML).toBe(cellInnerHTML) // no changes
        })

        test('handles invalid start position', () => {
            dom.highlightNode(cell, -1, 3)
            expect(cell.innerHTML).toBe(cellInnerHTML) // no changes
            dom.highlightNode(cell, cell.textContent!.length, 3)
            expect(cell.innerHTML).toBe(cellInnerHTML) // no changes
        })

        test('handles invalid length', () => {
            dom.highlightNode(cell, 0, 63) // length longer than cell.innerText
            expect(cell.innerHTML).toBe(cellInnerHTML) // no changes
            dom.highlightNode(cell, 22, 53) // length longer than characters between start and end
            expect(cell.innerHTML).toBe(cellInnerHTML) // no changes
        })

        test('highlights a single node', () => {
            dom.highlightNode(cell, 0, 1)
            const newCell =
                '<span style="color:#c0c5ce;"><span><span><span class="selection-highlight">\t</span></span></span></span><span style="color:#fff3bf;"><span>ServeHTTP</span></span><span style="color:#c0c5ce;"><span>(</span></span><span style="color:#c0c5ce;"><span>ResponseWriter</span></span><span style="color:#c0c5ce;"><span>,</span></span><span style="color:#c0c5ce;"><span> </span></span><span style="color:#329af0;"><span>*</span></span><span style="color:#c0c5ce;"><span>Request</span></span><span style="color:#c0c5ce;"><span>)</span></span>'
            expect(cell.innerHTML).toBe(newCell)
        })

        test('highlights multiple nodes', () => {
            dom.highlightNode(cell, 2, 2)
            dom.highlightNode(cell, 23, 2)
            const newCell =
                '<span style="color:#c0c5ce;"><span>\t</span></span><span style="color:#fff3bf;"><span><span>S<span class="selection-highlight">er</span>veHTTP</span></span></span><span style="color:#c0c5ce;"><span>(</span></span><span style="color:#c0c5ce;"><span><span>ResponseWrit<span class="selection-highlight">er</span></span></span></span><span style="color:#c0c5ce;"><span>,</span></span><span style="color:#c0c5ce;"><span> </span></span><span style="color:#329af0;"><span>*</span></span><span style="color:#c0c5ce;"><span>Request</span></span><span style="color:#c0c5ce;"><span>)</span></span>'
            expect(cell.innerHTML).toBe(newCell)
        })

        test('does not repeatedly highlight multiple nodes', () => {
            dom.highlightNode(cell, 0, 11)
            dom.highlightNode(cell, 0, 23)
            const newCell =
                '<span style="color:#c0c5ce;"><span><span><span class="selection-highlight">	</span></span></span></span><span style="color:#fff3bf;"><span><span><span class="selection-highlight">ServeHTTP</span></span></span></span><span style="color:#c0c5ce;"><span><span><span class="selection-highlight">(</span></span></span></span><span style="color:#c0c5ce;"><span><span><span class="selection-highlight">ResponseWrit</span>er</span></span></span><span style="color:#c0c5ce;"><span>,</span></span><span style="color:#c0c5ce;"><span> </span></span><span style="color:#329af0;"><span>*</span></span><span style="color:#c0c5ce;"><span>Request</span></span><span style="color:#c0c5ce;"><span>)</span></span>'
            expect(cell.innerHTML).toBe(newCell)
        })

        test('highlights after offset', () => {
            dom.highlightNode(cell, 2, 3)
            const newCell =
                '<span style="color:#c0c5ce;"><span>\t</span></span><span style="color:#fff3bf;"><span><span>S<span class="selection-highlight">erv</span>eHTTP</span></span></span><span style="color:#c0c5ce;"><span>(</span></span><span style="color:#c0c5ce;"><span>ResponseWriter</span></span><span style="color:#c0c5ce;"><span>,</span></span><span style="color:#c0c5ce;"><span> </span></span><span style="color:#329af0;"><span>*</span></span><span style="color:#c0c5ce;"><span>Request</span></span><span style="color:#c0c5ce;"><span>)</span></span>'
            expect(cell.innerHTML).toBe(newCell)
        })

        test('highlights entire cell', () => {
            dom.highlightNode(cell, 0, cell.textContent!.length)
            const newCell =
                '<span style="color:#c0c5ce;"><span><span><span class="selection-highlight">\t</span></span></span></span><span style="color:#fff3bf;"><span><span><span class="selection-highlight">ServeHTTP</span></span></span></span><span style="color:#c0c5ce;"><span><span><span class="selection-highlight">(</span></span></span></span><span style="color:#c0c5ce;"><span><span><span class="selection-highlight">ResponseWriter</span></span></span></span><span style="color:#c0c5ce;"><span><span><span class="selection-highlight">,</span></span></span></span><span style="color:#c0c5ce;"><span><span><span class="selection-highlight"> </span></span></span></span><span style="color:#329af0;"><span><span><span class="selection-highlight">*</span></span></span></span><span style="color:#c0c5ce;"><span><span><span class="selection-highlight">Request</span></span></span></span><span style="color:#c0c5ce;"><span><span><span class="selection-highlight">)</span></span></span></span>'
            expect(cell.innerHTML).toBe(newCell)
        })
    })
})
