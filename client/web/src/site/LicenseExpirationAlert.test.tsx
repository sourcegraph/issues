import React from 'react'
import { subMonths, addDays } from 'date-fns'
import renderer from 'react-test-renderer'
import { LicenseExpirationAlert } from './LicenseExpirationAlert'

describe('LicenseExpirationAlert.test.tsx', () => {
    test('expiring soon', () => {
        expect(
            renderer.create(<LicenseExpirationAlert expiresAt={addDays(new Date(), 3)} daysLeft={3} />)
        ).toMatchSnapshot()
    })

    test('expired', () => {
        expect(
            renderer.create(<LicenseExpirationAlert expiresAt={subMonths(new Date(), 3)} daysLeft={0} />)
        ).toMatchSnapshot()
    })
})
