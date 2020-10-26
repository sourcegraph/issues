/**
 * Highlights a node using recursive node walking.
 *
 * @param node the node to highlight
 * @param start the current character position (starts at 0).
 * @param length the number of characters to highlight.
 */
export function highlightNode(node: HTMLElement, start: number, length: number): void {
    if (start < 0 || length <= 0 || start >= node.textContent!.length) {
        return
    }

    // return if length is invalid/longer than remaining characters between start and end
    if (length > node.textContent!.length - start) {
        return
    }
    // We want to treat text nodes as walkable so they can be highlighted. Wrap these in a span and
    // replace them in the DOM.
    if (node.nodeType === Node.TEXT_NODE && node.textContent !== null) {
        const span = document.createElement('span')
        span.innerHTML = node.textContent
        node.parentNode!.replaceChild(span, node)
        node = span
    }
    node.classList.add('annotated-selection-match')
    highlightNodeHelper(node, 0, start, length)
}

interface HighlightResult {
    highlightingCompleted: boolean
    charsConsumed: number
    charsHighlighted: number
}

/**
 * Highlights a node using recursive node walking.
 *
 * @param currNode the current node being walked.
 * @param currOffset the current character position (starts at 0).
 * @param start the offset character where highlting starts.
 * @param length the number of characters to highlight.
 */
function highlightNodeHelper(
    currentNode: HTMLElement,
    currentOffset: number,
    start: number,
    length: number
): HighlightResult {
    if (length === 0) {
        return { highlightingCompleted: true, charsConsumed: 0, charsHighlighted: 0 }
    }

    const origOffset = currentOffset
    const numberChildNodes = currentNode.childNodes.length

    let charsHighlighted = 0

    for (let index = 0; index < numberChildNodes; ++index) {
        if (currentOffset >= start + length) {
            return { highlightingCompleted: true, charsConsumed: 0, charsHighlighted: 0 }
        }
        const isLastNode = index === currentNode.childNodes.length - 1
        const child = currentNode.childNodes[index]

        switch (child.nodeType) {
            case Node.TEXT_NODE: {
                const nodeText = child.textContent!

                if (currentOffset <= start && currentOffset + nodeText.length > start) {
                    // Current node overlaps start of highlighting.
                    child.remove()

                    // The characters beginning at the start of highlighting and extending to the end of the node.
                    const rest = nodeText.slice(start - currentOffset)

                    const containerNode = document.createElement('span')
                    if (nodeText.slice(0, Math.max(0, start - currentOffset))) {
                        // If characters were consumed leading up to the start of highlighting, add them to the parent.
                        containerNode.append(
                            document.createTextNode(nodeText.slice(0, Math.max(0, start - currentOffset)))
                        )
                    }

                    if (rest.length >= length) {
                        // The highlighted range is fully contained within the node.
                        if (currentNode.classList.contains('selection-highlight')) {
                            // Nothing to do; it's already highlighted.
                            currentNode.append(child)
                        } else {
                            const text = rest.slice(0, Math.max(0, length))
                            const highlight = document.createElement('span')
                            highlight.className = 'selection-highlight'
                            highlight.append(document.createTextNode(text))
                            containerNode.append(highlight)
                            if (rest.length > length) {
                                // There is more in the span than the highlighted chars.
                                containerNode.append(document.createTextNode(rest.slice(length)))
                            }

                            if (currentNode.childNodes.length === 0 || isLastNode) {
                                currentNode.append(containerNode)
                            } else {
                                currentNode.insertBefore(
                                    containerNode,
                                    currentNode.childNodes[index] || currentNode.firstChild
                                )
                            }
                        }

                        return { highlightingCompleted: true, charsConsumed: nodeText.length, charsHighlighted: length }
                    }

                    // Else the highlighted range spans multiple nodes.
                    charsHighlighted += rest.length

                    const highlight = document.createElement('span')
                    highlight.className = 'selection-highlight'
                    highlight.append(document.createTextNode(rest))
                    containerNode.append(highlight)

                    if (currentNode.childNodes.length === 0 || isLastNode) {
                        if (currentNode.classList.contains('selection-highlight')) {
                            // Nothing to do; it's already highlighted.
                            currentNode.append(child)
                        } else {
                            currentNode.append(containerNode)
                        }
                    } else {
                        currentNode.insertBefore(containerNode, currentNode.childNodes[index] || currentNode.firstChild)
                    }
                }

                currentOffset += nodeText.length
                break
            }

            case Node.ELEMENT_NODE: {
                const elementNode = child as HTMLElement
                const result = highlightNodeHelper(
                    elementNode,
                    currentOffset,
                    start + charsHighlighted,
                    length - charsHighlighted
                )
                if (result.highlightingCompleted) {
                    return result
                }
                currentOffset += result.charsConsumed
                charsHighlighted += result.charsHighlighted
                break
            }
        }
    }

    return { highlightingCompleted: false, charsConsumed: currentOffset - origOffset, charsHighlighted }
}
