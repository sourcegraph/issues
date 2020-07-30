import * as util from 'util'
import * as React from 'react'
import renderer from 'react-test-renderer'
import { ActivationProps } from '../../../shared/src/components/activation/Activation'
import * as GQL from '../../../shared/src/graphql/schema'
import { withActivation } from './withActivation'

const Component: React.FunctionComponent<ActivationProps & { authenticatedUser: GQL.User | null }> = (
    props: ActivationProps & { authenticatedUser: GQL.User | null }
) => <div>activation steps: {props.activation ? util.inspect(props.activation) : 'undefined'}</div>

describe.skip('withActivation', () => {
    const ComponentWithActivation = withActivation(Component)

    test('no user', () => {
        const component = renderer.create(<ComponentWithActivation authenticatedUser={null} />)
        expect(component.toJSON()).toMatchSnapshot()
    })
    test('user, admin', () => {
        const mockUser = {
            id: 'userID',
            username: 'username',
            email: 'user@me.com',
            siteAdmin: true,
        }
        const component = renderer.create(<ComponentWithActivation authenticatedUser={mockUser as GQL.User} />)
        expect(component.toJSON()).toMatchSnapshot()
    })
    test('user, non-admin', () => {
        const mockUser = {
            id: 'userID',
            username: 'username',
            email: 'user@me.com',
            siteAdmin: false,
        }
        const component = renderer.create(<ComponentWithActivation authenticatedUser={mockUser as GQL.User} />)
        expect(component.toJSON()).toMatchSnapshot()
    })
})
