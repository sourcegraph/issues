# Details

## Data freshness

Searches scoped to specific repositories are always up-to-date. Sourcegraph automatically fetches repository contents with any user action specific to the repository and makes new commits and branches available for searching and browsing immediately.

Unscoped search results over large repository sets may trail latest default branch revisions by some interval of time. This interval is a function of the number of repositories and the computational resources devoted to search indexing.

## Max file size

By default, files larger than 1 MB are excluded from search results. Use the [search.largeFiles](../../../admin/config/site_config.md#search-largeFiles) keyword to specify files to be indexed and searched regardless of size.

## Exclude files and directories

You can exclude files and directories from search by adding the file _.sourcegraph/ignore_ to
the root directory of your repository. Sourcegraph interprets each line in the _ignore_ file as globbing
pattern. Files or directories matching those patterns will not show up in the search results.

The _ignore_ file is tied to a commit. This means, if you committed an _ignore_ file to a 
feature branch but not to your default branch, then only search results for the feature branch
will be filtered while the default branch will show all results.

Example:
```
# .sourcegraph/ignore
# lines starting with # are comments and are ignored
# empty lines are ignored, too

# ignore the directory node_modules/
node_modules/

# ignore the directory src/data/
src/data/

# ** matches all characters, while * matches all characters except /
# ignore all JSON files
**.json

# ignore all JSON files at the root of the repository
*.json

# ignore all JSON files within the directory data/
data/**.json

# ignore all data folders
data/
**/data/

# ignore all files that start with numbers
[0-9]*.*
**/[0-9]*.*
```

Our syntax follows closely what is documented in 
[the linux documentation project](https://tldp.org/LDP/GNU-Linux-Tools-Summary/html/x11655.htm).
However, we distinguish between `*` and `**`: While `**` matches all characters, `*` matches all characters 
except the path separator `/`.

Note that invalid globbing patterns will cause an error and searches over commits containing a broken _ignore_ file 
will not return any result.
