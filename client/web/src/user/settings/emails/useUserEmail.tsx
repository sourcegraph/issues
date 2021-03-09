import { gql } from 'graphql-request'
import { Variables } from 'graphql-request/dist/types'
import { QueryObserverResult, useMutation, UseMutationResult, useQuery, useQueryClient } from 'react-query'
import fetcher from '../../../client'
import {
    AddUserEmailResult,
    RemoveUserEmailResult,
    ResendVerificationEmailResult,
    SetUserEmailVerifiedResult,
} from '../../../graphql-operations'

const ADD_USER_EMAIL = gql`
    mutation AddUserEmail($user: ID!, $email: String!) {
        addUserEmail(user: $user, email: $email) {
            alwaysNil
        }
    }
`
const GET_USER_EMAIL = gql`
    query UserEmails($user: ID!) {
        node(id: $user) {
            ... on User {
                emails {
                    email
                    isPrimary
                    verified
                    verificationPending
                    viewerCanManuallyVerify
                }
            }
        }
    }
`
const SET_USER_EMAIL_VERIFIED = gql`
    mutation SetUserEmailVerified($user: ID!, $email: String!, $verified: Boolean!) {
        setUserEmailVerified(user: $user, email: $email, verified: $verified) {
            alwaysNil
        }
    }
`
const RESEND_EMAIL_VERIFICATION = gql`
    mutation ResendVerificationEmail($user: ID!, $email: String!) {
        resendVerificationEmail(user: $user, email: $email) {
            alwaysNil
        }
    }
`
const REMOVE_EMAIL = gql`
    mutation RemoveUserEmail($user: ID!, $email: String!) {
        removeUserEmail(user: $user, email: $email) {
            alwaysNil
        }
    }
`

export const useRemoveUserEmail = (): UseMutationResult<RemoveUserEmailResult> => {
    const queryClient = useQueryClient()
    const mutation = useMutation((data: Variables) => fetcher({ queryKey: [REMOVE_EMAIL, data] }), {
        onSuccess: async (data, { user }) => {
            await queryClient.refetchQueries([GET_USER_EMAIL, { user }])
        },
    })

    return mutation as UseMutationResult<RemoveUserEmailResult>
}

export const useResendEmailVerification = (): UseMutationResult<ResendVerificationEmailResult> => {
    const queryClient = useQueryClient()
    const mutation = useMutation((data: Variables) => fetcher({ queryKey: [RESEND_EMAIL_VERIFICATION, data] }), {
        onSuccess: async (data, { user }) => {
            await queryClient.refetchQueries([GET_USER_EMAIL, { user }])
        },
    })

    return mutation as UseMutationResult<ResendVerificationEmailResult>
}

export const useSetUserEmailVerified = (): UseMutationResult<SetUserEmailVerifiedResult> => {
    const queryClient = useQueryClient()
    const mutation = useMutation((data: Variables) => fetcher({ queryKey: [SET_USER_EMAIL_VERIFIED, data] }), {
        onSuccess: async (data, { user }) => {
            await queryClient.refetchQueries([GET_USER_EMAIL, { user }])
        },
    })

    return mutation as UseMutationResult<SetUserEmailVerifiedResult>
}

export const useAddUserEmail = (): UseMutationResult<AddUserEmailResult> => {
    const queryClient = useQueryClient()
    const mutation = useMutation((data: Variables) => fetcher({ queryKey: [ADD_USER_EMAIL, data] }), {
        onMutate: async data => {
            await queryClient.cancelQueries([GET_USER_EMAIL, { user: data.user }])

            const previousUserEmail = queryClient.getQueryData([GET_USER_EMAIL, { user: data.user }])
            queryClient.setQueryData([GET_USER_EMAIL, { user: data.user }], (oldData: any) => ({
                node: { emails: [...oldData.node.emails, data] },
            }))
            return { previousUserEmail }
        },
    })
    console.log(mutation)
    return mutation as UseMutationResult<AddUserEmailResult>
}

export const useGetUserEmail = (userID: string): QueryObserverResult => {
    const result = useQuery<any>([GET_USER_EMAIL, { user: userID }], {
        select: data => data.node.emails,
    })

    return result
}
