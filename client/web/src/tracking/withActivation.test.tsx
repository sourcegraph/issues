import * as util from 'util'
import * as React from 'react'
import renderer from 'react-test-renderer'
import { ActivationProps } from '../../../shared/src/components/activation/Activation'
import { withActivation } from './withActivation'
import { AuthenticatedUser } from '../auth'

const Component: React.FunctionComponent<ActivationProps & { authenticatedUser: AuthenticatedUser | null }> = (
    props: ActivationProps & { authenticatedUser: AuthenticatedUser | null }
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
        const component = renderer.create(<ComponentWithActivation authenticatedUser={mockUser as AuthenticatedUser} />)
        expect(component.toJSON()).toMatchSnapshot()
    })
    test('user, non-admin', () => {
        const mockUser = {
            id: 'userID',
            username: 'username',
            email: 'user@me.com',
            siteAdmin: false,
        }
        const component = renderer.create(<ComponentWithActivation authenticatedUser={mockUser as AuthenticatedUser} />)
        expect(component.toJSON()).toMatchSnapshot()
    })
})
