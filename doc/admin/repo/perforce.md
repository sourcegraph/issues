# Using Perforce depots with Sourcegraph

Sourcegraph supports [Perforce](https://perforce.com) depots using the [git p4](https://git-scm.com/docs/git-p4) adapter. This creates an equivalent Git repository from a Perforce depot. For Sourcegraph <3.25.1, [`src serve-git`](../external_service/src_serve_git.md), Sourcegraph's tool for serving local directories, is required. For Sourcegraph 3.25.1+ an experimental feature can be enabled to configure Perforce depots through the Sourcegraph UI.

Screenshot of using Sourcegraph for code navigation in a Perforce depot:

![Viewing a Perforce repository on Sourcegraph](https://sourcegraphstatic.com/git-p4-example.png)


## Sourcegraph 3.25.1+ configuration instructions

Adding Perforce depots through the UI is an experimental feature in Sourcegraph 3.25.1. To access this functionality, a site admin must enable the experimental feature in the [site configuration](../config/site_config.md):

```json
{
	"experimentalFeatures": {
		"perforce": "enabled"
  }
  ...
}
```

To connect Perforce to Sourcegraph:

1. Depending on whether you are a site admin or user:
   1. *Site admin*: Go to **Site admin > Manage repositories > Add repositories**
   1. *User*: Go to **Settings > Manage repositories**.
1. Select **Perforce**.
1. Configure the connection to Perforce using the action buttons above the text field, and additional fields can be added using <kbd>Cmd/Ctrl+Space</kbd> for auto-completion. See the [configuration documentation below](#configuration).
1. Press **Add repositories**.

**NOTE** That adding code hosts as a user is currently in private beta.

### Depot syncing

> NOTE: Only "local" type depots are supported.

Use the `depots` field to configure which depots are mirrored/synchronized as Git repositories to Sourcegraph:

- [`depots`](perforce.md#depots)<br>A list of depot paths that can be either a depot root or an arbitrary subdirectory.
- [`p4.user`](perforce.md#p4-user)<br>The user to be authenticated for p4 CLI, and should be capable of performing `p4 ping`, `p4 login`, `p4 trust` and any p4 commands involved with `git p4 clone` and `git p4 sync` for listed `depots`. If repository permissions are mirrored, the user needs additional ability to perform the `p4 protects`, `p4 groups`, `p4 group`, `p4 users` commands (aka. "super" access level).
- [`p4.passwd`](perforce.md#p4-passwd)<br>The ticket value to be used for authenticating the `p4.user`. It is recommended to create tickets of users in a group that never expire. Use the command `p4 -u <p4.user> login -p -a` to obtain a ticket value.

Notable things about depot syncing:

- It takes approximately one second to import one Perforce change into a Git commit, this translates to sync a Perforce depot with 1000 changes takes approximately 1000 seconds, which is about 17 minutes. It is possible to limit the maximum changes to import using `maxChanges` config option.
- Rename of a Perforce depot will cause a re-import of the depot, including changing the depot on the Perforce server or the `repositoryPathPattern` config option.

### Repository permissions

> NOTE: Permissions syncing for Perforce depots is available in Sourcegraph 3.26+.

To enable permissions syncing for Perforce depots by including the `authorization` field:

```json
{
  ...
  "authorization": {}
}
```

Sourcegraph only supports repository-level permissions and does not match the granularity of Perforce access control lists (which supports file-level permissions). The workaround is for site admins to sync arbitrary subdirectories of a depot, which can then enforce permissions in Sourcegraph. We suggest using the most concrete path of your permissions boundary.

For example, if your Perforce depot `//Sourcegraph/` has different permissions for `//Sourcegraph/Backend/` and some subdirectories of `//Sourcegraph/Frontend/`, we recommend setting the following `depots`:

```json
{
  ...
  "depots": [
    "//Sourcegraph/Backend/",
    "//Sourcegraph/Frontend/Web/",
    "//Sourcegraph/Frontend/Extension/"
  ]
}
```

By configuring each subdirectory that has unique permissions, Sourcegraph is able to recognize and enforce permissions for each defined repository. You *cannot* define these permissions as:


```json
{
  ...
  "depots": [
    "//Sourcegraph/Backend/",
    "//Sourcegraph/Frontend/",
    "//Sourcegraph/Frontend/Extension/"
  ]
}
```

as this will override the permissions for the `//Sourcegraph/Frontend/Extension/` depot. See Known limitations below. 

#### Known limitations

Sourcegraph uses prefix-matching to determine if a user has access to a repository in Sourcegraph. That means if a user has access to a directory and also has exclusions to some subdirectories, _those exclusions will not be enforced in Sourcegraph_ because Sourcegraph does not support file-level permissions.

For example, consider the following output of `p4 protects -u alice`:

```text
list user * * -//...
list user * * -//spec/...
write user alice * //TestDepot/...
=write user alice * -//TestDepot/Secret/...
```

If the site admin configures `"depots": ["//TestDepot/"]`, the exclusion of the last line will not be enforced in Sourcegraph. In other words, the user alice _will have access_ to `//TestDepot/Secret/` in Sourcegraph even though alice does not have access to this directory on the Perforce Server. To mitigate, use the most concrete path of your permissions boundary as described in the above section.

Besides, the host (fourth) value is disregarded from the output of `p4 protects`.

### Configuration

<div markdown-func=jsonschemadoc jsonschemadoc:path="admin/external_service/perforce.schema.json">[View page on docs.sourcegraph.com](https://docs.sourcegraph.com/admin/external_service/perforce) to see rendered content.</div>

## Sourcegraph <3.25.1 configuration instructions with `src serve-git`

### Prerequisites

- Git
- Perforce `p4` CLI configured to access your Perforce depot
- `git p4` (see "[Adding `git p4` to an existing install](https://git.wiki.kernel.org/index.php/GitP4#Adding_git-p4_to_an_existing_install)")
- [`src serve-git`](../external_service/src_serve_git.md)

### Create an equivalent Git repository and serve it to Sourcegraph

For each Perforce repository you want to use with Sourcegraph, follow these steps:

1. Create a local Git repository with the contents of your Perforce depot: `git p4 clone //DEPOT/PATH@all` (replace `//DEPOT/PATH` with the Perforce repository path).
1. Run `src serve-git` from the parent directory that holds all of the new local Git repositories.
1. Follow the instructions in the [`src serve-git` Quickstart](../external_service/src_serve_git.md#quickstart) to add the repositories to your Sourcegraph instance.

### Updating Perforce depots

To update the repository after new Perforce commits are made, run `git p4 sync` in the local repository directory. These changes will be automatically reflected in Sourcegraph as long as `src serve-git` is running.

We recommend running this command on a periodic basis using a cron job, or some other scheduler. The frequency will dictate how fresh the code is in Sourcegraph, and can range from once every 10s to once per day, depending on how large your codebase is and how long it takes `git p4 sync` to complete.

### Alternative to `src serve-git`: push the new Git repository to a code host

If you prefer, you can skip using `src serve-git`, and instead push the new local Git repository to a Git-based code host of your choice. For updates, you would run `git p4 sync && git push` periodically.

If you do this, the repositories you created on your Git host are normal Git repositories, so you can [add the repositories to Sourcegraph](index.md) as you would any other Git repositories.

### Alternative for extra-large codebases

The instructions below will help you get Perforce depots on Sourcegraph quickly and easily, while retaining all code change history. If your Perforce codebase is large enough that converting it to Git takes long enough to cause noticeable staleness on Sourcegraph, you can use `src-expose`'s [optional syncing functionality](../external_service/non-git.md#syncing-repositories) along with a faster fetching command (like `p4 sync` instead of `git p4 sync`) to periodically fetch and squash changes without trying to preserve the original Perforce history.

## Known issues

We intend to improve Sourcegraph's Perforce support in the future. Please [file an issue](https://github.com/sourcegraph/sourcegraph/issues) to help us prioritize any specific improvements you'd like to see.

- Sourcegraph was initially built for Git repositories only, so it exposes Git concepts that are meaningless for converted Perforce depots, such as the commit SHA, branches, and tags.
- The commit messages for a Perforce depot converted to a Git repository have an extra line at the end with Perforce information, such as `[git-p4: depot-paths = "//guest/acme_org/myproject/": change = 12345]`.
