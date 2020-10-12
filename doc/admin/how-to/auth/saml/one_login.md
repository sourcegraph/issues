# Configuring SAML with One Login

## 1. Create a SAML app in OneLogin

1. Go to https://mycompany.onelogin.com/apps/find (replace "mycompany" with your company's OneLogin
   ID).
1. Select "SAML Test Connector (SP)" and click "Save".
1. Under the "Configuration" tab, set the following properties (replacing `https://sourcegraph.example.com` with your Sourcegraph URL):
   * `Audience`:  https://sourcegraph.example.com/.auth/saml/metadata
   * `Recipient`: https://sourcegraph.example.com/.auth/saml/acs
   * `ACS (Consumer) URL Validator`: https:<span>//</span>sourcegraph\\.example\\.com\\/\\.auth\\/saml\\/acs<br>
     (This is regular expression that matches the URL `https://sourcegraph.example.com/.auth/saml/acs`)
   * `ACS (Consumer) URL`: https://sourcegraph.example.com/.auth/saml/acs
1. Under the "Parameters" tab, ensure the following parameters exist:<br>
   * Email (NameID): Email
   * DisplayName:    First Name         Include in SAML Assertion: ✓
   * login:          AD user name       Include in SAML Assertion: ✓
1. Save the app in OneLogin.
1. Find the Issuer URL in the OneLogin app configuration page, under the "SSO" tab, under "Issuer
   URL". It should look something like `https://mycompany.onelogin.com/saml/metadata/123456` or
   `https://app.onelogin.com/saml/metadata/123456`. Record this for the next section.

## 2. Add the SAMl auth provider to Sourcegraph site config

1. In Sourcegraph [site config](../../config/site_config.md), ensure `externalURL` is set the same Sourcegraph URL you used in the previous section (i.e., what you replaced `https://sourcegraph.example.com` with). Be mindful to use the exact same scheme (`http` or `https`), and there should be no trailing slash.
1. Add an item to `auth.providers` with `type` "saml" and `identityProviderMetadataURL` set to the
   Issuer URL recorded from the previous section. Here is an example:

```json
{
 // ...
 "externalURL": "https://sourcegraph.example.com",
 "auth.providers": [
   {
     "type": "saml",
     "identityProviderMetadataURL": "<issuer URL>"
   }
 ]
}
```

Confirm there are no error messages in the `sourcegraph/server` Docker container logs (or the
`sourcegraph-frontend` pod logs, if Sourcegraph is deployed to a Kubernetes cluster). The most
likely error message indicating a problem is `Error prefetching SAML service provider metadata`. See
[SAML troubleshooting](../saml.md#saml-troubleshooting) for more tips.
