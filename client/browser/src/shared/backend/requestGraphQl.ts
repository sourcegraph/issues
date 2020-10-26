import { from } from 'rxjs'
import { requestGraphQLCommon } from '../../../../shared/src/graphql/graphql'
import { background } from '../../browser-extension/web-extension-api/runtime'
import { isBackground } from '../context'

/**
 * Returns a platform-appropriate implementation of the function used to make requests to our GraphQL API.
 *
 * In the browser extension, the returned function will make all requests from the background page.
 *
 * In the native integration, the returned function will rely on the `requestGraphQL` implementation from `/shared`.
 */
export const requestGraphQlHelper = (isExtension: boolean, baseUrl: string) => <T, V = object>({
    request,
    variables,
}: {
    request: string
    variables: V
}) =>
    isExtension && !isBackground
        ? from(
              background.requestGraphQL<T, V>({ request, variables })
          )
        : requestGraphQLCommon<T, V>({
              request,
              variables,
              baseUrl,
              credentials: 'include',
          })
