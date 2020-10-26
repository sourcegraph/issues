# Product documentation implementation

The [documentation guidelines](https://about.sourcegraph.com/handbook/engineering/product_documentation) apply to product documentation. This page has information specific to this repository's documentation.

## Documentation directory structure

The documentation is broken down into 3 different areas:

1. User
1. Admin
1. Development

Each of these areas has the docs organized by the 4 different types:

1. Tutorials
1. How-to guides
1. Explanation or background information
1. Reference

This structure is inspired by the [Divio documentation system](https://documentation.divio.com/).

## Previewing changes locally

You can preview the documentation site at http://localhost:5080 when running Sourcegraph in [local development](../getting-started/index.md) (using `dev/start.sh` or `enterprise/dev/start.sh`). It uses content, templates, and assets from the local disk. There is no caching or background build process, so you'll see all changes reflected immediately after you reload the page in your browser.

You can also run the docsite on its own with the following command:

```sh
./dev/docsite.sh -config doc/docsite.json serve -http=localhost:5080
```

## Linking to documentation in-product

In-product documentation links should point to `/help/PATH` instead of using an absolute URL of the form https://docs.sourcegraph.com/PATH. This ensures they link to the documentation for the current product version. There is a redirect (when using either `<a>` or react-router `<Link>`) from `/help/PATH` to the versioned docs.sourcegraph.com URL (https://docs.sourcegraph.com/@VERSION/PATH).

## Adding images to the documentation

We generally try to avoid adding large binary files to our repository. Images to be used in documentation fall under that category, but there can be exceptions if the images are small.

- If the image is less than 100kb in size, it can be added to the `./doc` folder.
- If it is bigger than 100kb, upload it to the [sourcegraph-assets/docs/images](https://console.cloud.google.com/storage/browser/sourcegraph-assets/docs/images/?project=sourcegraph-de&folder=true&organizationId=true) on Google Cloud storage and link to it.

## Updating documentation

To update documentation content, templates, or assets on https://docs.sourcegraph.com, push changes in the `doc/` directory to this repository's `main` branch, then wait up to 5 minutes. Every 5 minutes, docs.sourcegraph.com reloads all content, templates, and assets from `main`.

- Documentation content lives in `doc/**/*.md`.
- The sidebar lives in `doc/sidebar.md`. Only important pages belong in the sidebar; use section index page links for other documents.
- Assets and templates live in `doc/_resources/{templates,assets}`.

## Advanced documentation site

Our documentation site (https://docs.sourcegraph.com) runs [docsite](https://github.com/sourcegraph/docsite).

See "[Updating documentation](#updating-documentation)" and "[Previewing changes locally](#previewing-changes-locally)" for the most common workflows involving the documentation site.

## Forcing immediate reload of data

The docs.sourcegraph.com site reloads content, templates, and assets every 5 minutes. After you push a [documentation update](#updating-documentation), just wait up to 5 minutes to see your changes reflected on docs.sourcegraph.com.

If you can't wait 5 minutes and need to force a reload, you can kill the `docs-sourcegraph-com-*` Kubernetes pod on the Sourcegraph.com Kubernetes cluster. (It will restart and come back online with the latest data.)

## Other ways of previewing changes locally (very rare)

The [local documentation server](#previewing-changes-locally) on http://localhost:5080 only serves a single version of the documentation (from the `doc/` directory of your working tree). This usually suffices.

In very rare cases, you may want to run a local documentation server with a different configuration (described in the following sections).

<!-- TODO(ryan): Uncomment once https://github.com/sourcegraph/docsite/issues/13 is fixed.

### Running multi-version support locally

> NOTE: The below does not currently work due to an issue with docsite being unable to load a combination of content and templates/assets locally and over http.

If you're working on a docs template change involving multiple content versions (i.e., doc site URL paths like `/@v1.2.3/my/doc/page`), then you must run a [docsite](https://github.com/sourcegraph/docsite) server that can read multiple content versions:

``` shell
DOCSITE_CONFIG=$(cat <<-'DOCSITE'
{
  "templates": "_resources/templates",
  "content": "https://codeload.github.com/sourcegraph/sourcegraph/zip/refs/heads/$VERSION#*/doc/",
  "baseURLPath": "/",
  "assets": "_resources/assets",
  "assetsBaseURLPath": "/assets/"
}
DOCSITE
) docsite serve -http=localhost:5081

```

This runs a docsite server on http://localhost:5081 that reads templates and assets from disk (so yo can see your changes reflected immediately upon page reload) but reads content from the remote Git repository at any version (by default `master` if no version is given in the URL path, as in `/@v1.2.3/my/doc/page`).
-->

### Running a local server that mimics prod configuration

If you want to run the doc site *exactly* as it's deployed (reading templates and assets from the remote Git repository, too), consult the current Kubernetes deployment spec and invoke `docsite serve` with the deployment's `DOCSITE_CONFIG` env var, the end result looking something like:

```bash
DOCSITE_CONFIG=$(cat <<-'DOCSITE'
{
  "templates": "https://codeload.github.com/sourcegraph/sourcegraph/zip/main#*/doc/_resources/templates/",
  "assets": "https://codeload.github.com/sourcegraph/sourcegraph/zip/main#*/doc/_resources/assets/",
  "content": "https://codeload.github.com/sourcegraph/sourcegraph/zip/refs/heads/$VERSION#*/doc/",
  "defaultContentBranch": "main",
  "baseURLPath": "/",
  "assetsBaseURLPath": "/assets/"
}
DOCSITE
) docsite serve -http=localhost:5081
```
