import React from 'react'
import { CampaignsDotComPage } from './CampaignsDotComPage'
import { mount } from 'enzyme'

describe('CampaignsDotComPage', () => {
    test('renders', () => {
        expect(mount(<CampaignsDotComPage />)).toMatchSnapshot()
    })
})
