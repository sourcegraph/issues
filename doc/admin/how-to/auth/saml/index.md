# SAML

Select your SAML identity provider for setup instructions:

- [Okta](okta.md)
- [Azure Active Directory (Azure AD)](azure_ad.md)
- [Microsoft Active Directory Federation Services (ADFS)](microsoft_adfs.md)
- [Auth0](generic.md)
- [OneLogin](one_login.md)
- [Ping Identity](generic.md)
- [Salesforce Identity](generic.md)
- [JumpCloud](jump_cloud.md)
- [Other](generic.md)

For advanced SAML configuration options, see the [`saml` auth provider documentation](../../config/site_config.md#saml).

> NOTE: Sourcegraph currently supports at most 1 SAML auth provider at a time (but you can configure additional auth providers of other types). This should not be an issue for 99% of customers.

### SAML troubleshooting

Setting the env var `INSECURE_SAML_LOG_TRACES=1` on the `sourcegraph/server` Docker container (or the `sourcegraph-frontend` pod if Sourcegraph is deployed to a Kubernetes cluster) causes all SAML requests and responses to be logged.
