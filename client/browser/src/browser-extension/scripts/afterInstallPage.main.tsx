// We want to polyfill first.
import '../../shared/polyfills'

import React from 'react'
import { render } from 'react-dom'
import { AfterInstallPageContent } from '../after-install-page/AfterInstallPageContent'
import { ThemeWrapper } from '../ThemeWrapper'

const AfterInstallPage: React.FunctionComponent = () => <ThemeWrapper>{AfterInstallPageContent}</ThemeWrapper>

render(<AfterInstallPage />, document.querySelector('#root'))
