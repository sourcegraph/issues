import 'focus-visible'

import { configureActions } from '@storybook/addon-actions'
import { withConsole } from '@storybook/addon-console'
import { withKnobs } from '@storybook/addon-knobs'
import { setLinkComponent, AnchorLink } from '../client/shared/src/components/Link'
import { withDesign } from 'storybook-addon-designs'
import isChromatic from 'chromatic/isChromatic'

export const decorators = [withKnobs, withDesign, (storyFn, context) => withConsole()(storyFn)(context)]

setLinkComponent(AnchorLink)

if (isChromatic()) {
  const style = document.createElement('style')
  style.innerHTML = `
      .monaco-editor .cursor {
        visibility: hidden !important;
      }
    `
  document.head.append(style)
}

configureActions({ depth: 100, limit: 20 })
