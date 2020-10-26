import React from 'react'
import * as GQL from '../../../../../../shared/src/graphql/schema'
import renderer from 'react-test-renderer'
import { SiteAdminProductLicenseNode } from './SiteAdminProductLicenseNode'
import { MemoryRouter } from 'react-router'

jest.mock('../../../dotcom/productSubscriptions/AccountName', () => ({ AccountName: 'AccountName' }))

describe('SiteAdminProductLicenseNode', () => {
    test('active', () => {
        expect(
            renderer
                .create(
                    <MemoryRouter>
                        <SiteAdminProductLicenseNode
                            node={
                                {
                                    createdAt: '2020-01-01',
                                    id: 'l1',
                                    licenseKey: 'lk1',
                                    info: {
                                        __typename: 'ProductLicenseInfo',
                                        expiresAt: '2021-01-01',
                                        productNameWithBrand: 'NB',
                                        tags: ['a'],
                                        userCount: 123,
                                    },
                                    subscription: {
                                        name: 's',
                                        activeLicense: { id: 'l1' },
                                        urlForSiteAdmin: '/s',
                                    } as GQL.IProductSubscription,
                                } as GQL.IProductLicense
                            }
                            showSubscription={true}
                        />
                    </MemoryRouter>
                )
                .toJSON()
        ).toMatchSnapshot()
    })

    test('inactive', () => {
        expect(
            renderer
                .create(
                    <MemoryRouter>
                        <SiteAdminProductLicenseNode
                            node={
                                {
                                    createdAt: '2020-01-01',
                                    id: 'l1',
                                    licenseKey: 'lk1',
                                    info: {
                                        __typename: 'ProductLicenseInfo',
                                        expiresAt: '2021-01-01',
                                        productNameWithBrand: 'NB',
                                        tags: ['a'],
                                        userCount: 123,
                                    },
                                    subscription: {
                                        name: 's',
                                        activeLicense: { id: 'l0' },
                                        urlForSiteAdmin: '/s',
                                    } as GQL.IProductSubscription,
                                } as GQL.IProductLicense
                            }
                            showSubscription={true}
                        />
                    </MemoryRouter>
                )
                .toJSON()
        ).toMatchSnapshot()
    })
})
