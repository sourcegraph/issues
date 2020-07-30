import { from } from 'rxjs'
import { requestGraphQL } from '../../../../shared/src/graphql/graphql'
import { background } from '../../browser-extension/web-extension-api/runtime'

/**
 * Returns a platform-appropriate implementation of the function used to make requests to our GraphQL API.
 *
 * In the browser extension, the returned function will make all requests from the background page.
 *
 * In the native integration, the returned function will rely on the `requestGraphQL` implementation from `/shared`.
 */
export const requestGraphQlHelper = (isExtension: boolean, baseUrl: string) => <T>({
    request,
    variables,
}: {
    request: string
    variables: {}
}) =>
    isExtension
        ? from(
              background.requestGraphQL<T>({ request, variables })
          )
        : requestGraphQL<T>({
              request,
              variables,
              baseUrl,
              credentials: 'include',
          })
