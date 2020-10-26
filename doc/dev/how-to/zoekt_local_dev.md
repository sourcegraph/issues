# Set up local development with Zoekt and Sourcegraph

```
$ git clone https://github.com/sourcegraph/sourcegraph
$ git clone https://github.com/sourcegraph/zoekt
```

To see your Zoekt changes reflected live on your local Sourcegraph instance, you'll have to modify your Sourcegraph `go.mod` file so that the Zoekt dependency is pointed to your local folder.

Change at the bottom:

```
replace (
    ...
    github.com/google/zoekt => <your zoekt repository directory>
    ...
)
```

This isn't hot reloaded so you might have to restart Sourcegraph on every Zoekt change. It may make sense to ensure your changes in Zoekt are working first before trying them out in Sourcegraph.

## Notes

Here are some commands you can run against Zoekt.

**Setup**

```
$ go install ./cmd/...
$ go install ./cmd/<specific command> (ex zoekt-archive-index)
```

The components that Sourcegraph uses from Zoekt are `zoekt-archive-index`, `zoekt-git-index`, `zoekt-sourcegraph-indexserver`, and `zoekt-webserver`.

```
# Direct usage
$ zoekt-index <repository>
$ zoekt <query>
```

**Misc**

Index files are stored in:
- `~/.zoekt` (zoekt cmd)
- `~/.sourcegraph/zoekt/index-X` (sourcegraph)

Local Sourcegraph Zoekt UI can be accesed at localhost:3070 and localhost:3071 (we have multiple instances because of horizontal scaling).
