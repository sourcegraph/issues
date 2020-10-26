import React from 'react'
import renderer from 'react-test-renderer'
import { BrandLogo } from './BrandLogo'

describe('BrandLogo', () => {
    test('renders light sourcegraph logo', () =>
        expect(
            renderer.create(<BrandLogo assetsRoot="/assets" isLightTheme={true} variant="logo" />).toJSON()
        ).toMatchSnapshot())

    test('renders light sourcegraph symbol', () =>
        expect(
            renderer.create(<BrandLogo assetsRoot="/assets" isLightTheme={true} variant="symbol" />).toJSON()
        ).toMatchSnapshot())

    test('renders dark sourcegraph logo', () =>
        expect(
            renderer.create(<BrandLogo assetsRoot="/assets" isLightTheme={false} variant="logo" />).toJSON()
        ).toMatchSnapshot())

    test('renders dark sourcegraph symbol', () =>
        expect(
            renderer.create(<BrandLogo assetsRoot="/assets" isLightTheme={false} variant="symbol" />).toJSON()
        ).toMatchSnapshot())

    test('renders light custom branding logo', () =>
        expect(
            renderer
                .create(
                    <BrandLogo
                        branding={{ brandName: 'b', light: { logo: 'l' } }}
                        assetsRoot="/assets"
                        isLightTheme={true}
                        variant="logo"
                    />
                )
                .toJSON()
        ).toMatchSnapshot())

    test('renders light custom branding symbol', () =>
        expect(
            renderer
                .create(
                    <BrandLogo
                        branding={{ brandName: 'b', light: { symbol: 's' } }}
                        assetsRoot="/assets"
                        isLightTheme={true}
                        variant="symbol"
                    />
                )
                .toJSON()
        ).toMatchSnapshot())

    test('renders dark custom branding logo', () =>
        expect(
            renderer
                .create(
                    <BrandLogo
                        branding={{ brandName: 'b', dark: { logo: 'l' } }}
                        assetsRoot="/assets"
                        isLightTheme={false}
                        variant="logo"
                    />
                )
                .toJSON()
        ).toMatchSnapshot())

    test('renders dark custom branding symbol', () =>
        expect(
            renderer
                .create(
                    <BrandLogo
                        branding={{ brandName: 'b', dark: { symbol: 's' } }}
                        assetsRoot="/assets"
                        isLightTheme={false}
                        variant="symbol"
                    />
                )
                .toJSON()
        ).toMatchSnapshot())
})
