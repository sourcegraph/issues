import React from 'react'
import { render } from '@testing-library/react'
import { StatusBar } from './StatusBar'
import { StatusBarItemWithKey } from '../../../../shared/src/api/extension/api/codeEditor'
import { BehaviorSubject } from 'rxjs'
import { extensionsController } from '../../../../shared/src/util/searchTestHelpers'

describe('StatusBar', () => {
    it('renders correctly', () => {
        const getStatusBarItems = () =>
            new BehaviorSubject<StatusBarItemWithKey[]>([
                { key: 'codecov', text: 'Coverage: 96%' },
                { key: 'code-owners', text: '2 code owners', tooltip: 'Code owners: @felixbecker, @beyang' },
            ]).asObservable()

        expect(
            render(<StatusBar getStatusBarItems={getStatusBarItems} extensionsController={extensionsController} />)
                .baseElement
        ).toMatchSnapshot()
    })

    // executes commands

    // empty state (no extensions)
})
