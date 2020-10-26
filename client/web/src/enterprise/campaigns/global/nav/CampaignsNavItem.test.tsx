import React from 'react'
import { CampaignsNavItem } from './CampaignsNavItem'
import { mount } from 'enzyme'
import { MemoryRouter } from 'react-router'

describe('CampaignsNavItem', () => {
    test('renders', () => {
        expect(
            mount(
                <MemoryRouter>
                    <CampaignsNavItem className="123" />
                </MemoryRouter>
            )
        ).toMatchSnapshot()
    })
})
