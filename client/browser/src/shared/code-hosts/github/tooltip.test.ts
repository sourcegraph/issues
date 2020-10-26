import { setElementTooltip } from './tooltip'

describe('setElementTooltip', () => {
    test('sets', () => {
        const template = document.createElement('span')
        template.classList.add('foo')
        template.append('bar')
        setElementTooltip(template, 'tt')
        expect(template.outerHTML).toBe('<span class="foo tooltipped tooltipped-n" aria-label="tt">bar</span>')
    })

    test('removes', () => {
        const template = document.createElement('span')
        template.classList.add('foo', 'tooltipped', 'tooltipped-n')
        template.setAttribute('aria-label', 'tt')
        template.append('bar')
        setElementTooltip(template, null)
        expect(template.outerHTML).toBe('<span class="foo">bar</span>')
    })
})
