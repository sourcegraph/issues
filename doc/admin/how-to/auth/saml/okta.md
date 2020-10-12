# Configuring SAML with Okta

## 1. Add a SAML application in Okta

1. Navigate to the "Classic UI" in the Okta Admin site. In the upper left-hand corner, it should say "Classic UI". If it says "Developer Console", click it and select "Classic UI". ![Okta Developer Console: Classic UI](https://user-images.githubusercontent.com/1646931/71300638-7a52fd80-234b-11ea-90cf-960820d4d5f2.png)
1. Go to the Applications tab. Click "Add Application" and then "Create New App". Select "Web" as the choice of Platform and "SAML 2.0" as the Sign on method. Then click "Create". ![Add application](https://user-images.githubusercontent.com/1646931/71300683-02390780-234c-11ea-8cbb-7c9987d3b472.png)
1. Give your app a name ("Sourcegraph") and click "Next".
1. Set the following values in the SAML Settings (replacing `https://sourcegraph.example.com` with your Sourcegraph URL):
  * **Single sign on URL:** `https://sourcegraph.example.com/.auth/saml/acs`<br>
    (Check the box for "Use this for Recipient URL and Destination URL")
  * **Audience URI (SP Entity ID):** `https://sourcegraph.example.com/.auth/saml/metadata`
  * **Attribute statements:**:<br>
    `email` (required): user.email<br>
    `login` (optional): user.login<br>
    `displayName` (optional): user.firstName<br>
1. Click "Next".
1. Select "I'm an Okta customer adding an internal app" and click "Finish".
1. In the Settings panel on the next page, find the "Identity Provider metadata" link and record its URL. ![Identity Provider metadata link](https://user-images.githubusercontent.com/1646931/71300825-63ada600-234d-11ea-858a-a489d8a79168.png)
1. Grant users or groups sign-in access in the "Assignments" tab. You can do other users later, but at the very least, grant your own Okta user access to the application, or else you won't be able to sign in.

## 2. Add the SAML auth provider to Sourcegraph site config

1. In Sourcegraph [site config](../../config/site_config.md), ensure `externalURL` is set the same Sourcegraph URL you used in the previous section (i.e., what you replaced `https://sourcegraph.example.com` with). Be mindful to use the exact same scheme (`http` or `https`), and there should be no trailing slash.
1. Add an item to `auth.providers` with `type` "saml" and `identityProviderMetadataURL` set to the URL you copied from the "Identity Provider metadata" link in the previous section. Here is an example of what your site configuration should look like:

```json
{
 // ...
 "externalURL": "https://sourcegraph.example.com",
 "auth.providers": [
   {
     "type": "saml",
     "identityProviderMetadataURL": "https://okta.example.com/app/8VglnckX0yyhdkp0bk00/sso/saml/metadata"
   }
 ]
}
```

Confirm there are no error messages in the `sourcegraph/server` Docker container logs (or the `sourcegraph-frontend` pod logs, if Sourcegraph is deployed to a Kubernetes cluster). The most likely error message indicating a problem is `Error prefetching SAML service provider metadata`. See [SAML troubleshooting](../saml.md#saml-troubleshooting) for more tips.
