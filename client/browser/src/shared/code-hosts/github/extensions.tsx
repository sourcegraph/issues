import { querySelectorOrSelf } from '../../util/dom'
import { MountGetter } from '../shared/codeHost'

export const getCommandPaletteMount: MountGetter = (container: HTMLElement): HTMLElement | null => {
    const className = 'command-palette-button'
    // This selector matches both GitHub Enterprise and github.com
    const existing = container.querySelector<HTMLElement>(`.Header .${className}`)
    if (existing) {
        return existing
    }
    // Legacy header (GitHub Enterprise)
    const gheHeaderElement = querySelectorOrSelf(container, '.HeaderMenu > :last-child')
    if (gheHeaderElement) {
        const mount = document.createElement('div')
        mount.classList.add(className)
        gheHeaderElement.prepend(mount)
        return mount
    }
    // github.com doesn't use HeaderMenu to wrap the right-hand-side menu anymore,
    // it has a flatter DOM structure
    // Instead of finding the parent to insert into, find the sibling to insert next to
    let rightNeighbor = querySelectorOrSelf(container, '.Header-item:nth-last-child(2)')
    if (rightNeighbor) {
        // Caveat: there is no noticiations icon if web notifications are disabled,
        // but the empty header item is still there
        if (rightNeighbor.previousElementSibling!.children.length !== 0) {
            rightNeighbor = rightNeighbor.previousElementSibling!
        }
        const mount = document.createElement('div')
        mount.classList.add('Header-item', 'mr-0', 'mr-lg-3', className)
        rightNeighbor.before(mount)
        return mount
    }
    return null
}
