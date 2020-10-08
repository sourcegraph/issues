# Saved searches

Saved searches lets you save and describe search queries so you can easily monitor the results on an ongoing basis. You can create a saved search for anything, including diffs and commits across all branches of your repositories.

Saved searches can be an early warning system for common problems in your code--and a way to monitor best practices, the progress of refactors, etc. Alerts for saved searches can be sent through email, ensuring you're aware of important code changes.

## Creating saved searches

A saved search consists of a description and a query, both of which you can define and edit.

Saved searches are written as JSON entries in settings, and they can be associated with a user or an org:

- User saved searches are only visible to (and editable by) the user that created them.
- Org saved searches are visible to (and editable by) all members of the org.

To create a saved search:

1. Go to **User menu > Saved searches** in the top navigation bar.
1. Press the **+ Add new search** button.
1. In the **Query** field, type in the components of the search query.
1. In the **Description** field, type in a human-readable description for your saved search.
1. In the user and org dropdown menu, select where you'd like the search to be saved.
1. Click **Create**. The saved search is created, and you can see the number of results.

Alternatively, to create a saved search from a search you've already run:

1. Execute a search from the homepage or navigation bar.
1. Press the **Save this search query** button that appears on the right side of the screen above the first result.
1. Follow the instructions from above to fill in the remaining fields.

To view saved searches, go to **User menu > Saved searches** in the top navigation bar.

## Configuring email notifications

Sourcegraph can automatically run your saved searches and notify you when new results are available via email. With this feature you can get notified about issues in your code (such as licensing issues, security changes, potential secrets being committed, etc.)

To configure email notifications, click **Edit** on a saved search and check the **Email notifications** checkbox and press **Save**. You will receive a notification telling you it is set up and working almost instantly!

By default, email notifications notify the owner of the configuration (either a single user or the entire org).

## Example saved searches

See the [search examples page](examples.md) for a useful list of searches to save.
