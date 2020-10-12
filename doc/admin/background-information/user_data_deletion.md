# User data deletion

As a site administrator, you have the ability to delete users and their associated data on the **Admin** -> **Users** page (https://sourcegraph.example.com/site-admin/users).

On this page, you are presented two options:

- Deleting a user: the user and ALL associated data is marked as deleted in the DB and never served again. You could undo this by running DB commands manually.
- Nuking a user, the user and ALL associated data is deleted forever (you CANNOT undo this).

When deleting or nuking a user, the following information is removed:

- All user data (access tokens, email addresses, external account info, survey responses, etc)
- Organization membership information (which organizations the user is a part of, any invitations created by or targeting the user).
- Sourcegraph extensions published by the user on the instance the deletion request is sent to.
- User, Organization, or Global settings authored or modified by the user.
