import { NEVER, of } from 'rxjs'
import { first } from 'rxjs/operators'
import { ContributableViewContainer } from '../protocol'
import { assertToJSON, integrationTestContext } from './testHelpers'

describe('Views (integration)', () => {
    describe('app.createPanelView', () => {
        test('no component', async () => {
            const { extensionAPI, services } = await integrationTestContext()
            const panelView = extensionAPI.app.createPanelView('p')
            panelView.title = 't'
            panelView.content = 'c'
            panelView.priority = 3
            await extensionAPI.internal.sync()

            const values = await services.panelViews
                .getPanelViews(ContributableViewContainer.Panel)
                .pipe(first(views => views.length > 0))
                .toPromise()
            assertToJSON(values, [
                {
                    id: 'p',
                    title: 't',
                    content: 'c',
                    priority: 3,
                    container: ContributableViewContainer.Panel,
                },
            ])
        })

        test('with component (location provider)', async () => {
            const { extensionAPI, services } = await integrationTestContext()

            const LOCATION_PROVIDER_ID = 'x'
            extensionAPI.languages.registerLocationProvider(LOCATION_PROVIDER_ID, ['*'], {
                provideLocations: () => NEVER,
            })

            const panelView = extensionAPI.app.createPanelView('p')
            panelView.title = 't'
            panelView.content = 'c'
            panelView.priority = 3
            panelView.component = { locationProvider: LOCATION_PROVIDER_ID }

            const values = await services.panelViews
                .getPanelViews(ContributableViewContainer.Panel)
                .pipe(first(views => views.length > 0))
                .toPromise()
            assertToJSON(
                values.map(view => ({ ...view, locationProvider: 'value not checked' })),
                [
                    {
                        id: 'p',
                        title: 't',
                        content: 'c',
                        priority: 3,
                        container: ContributableViewContainer.Panel,
                        locationProvider: 'value not checked',
                    },
                ]
            )
        })
    })

    test('app.registerViewProvider', async () => {
        const { extensionAPI, services } = await integrationTestContext()

        extensionAPI.app.registerViewProvider('v', {
            where: ContributableViewContainer.GlobalPage,
            provideView: parameters => of({ title: `t${parameters.x}`, content: [] }),
        })

        const view = await services.view
            .get('v', { x: 'y' })
            .pipe(first(view => view !== null))
            .toPromise()
        expect(view).toEqual({ title: 'ty', content: [] })
    })
})
