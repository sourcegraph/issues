# Code search

> → See the query [**syntax reference**](queries.md) and [**language reference**](language.md). See [search examples](examples.md) for inspiration.

[A recently published research paper from Google](https://static.googleusercontent.com/media/research.google.com/en//pubs/archive/43835.pdf) and a [Google developer survey](https://docs.google.com/document/d/1LQxLk4E3lrb3fIsVKlANu_pUjnILteoWMMNiJQmqNVU/edit#heading=h.xxziwxixfqq3) showed that 98% of developers consider their Sourcegraph-like internal code search tool to be critical, and developers use it on average for 5.3 sessions each day, primarily to (in order of frequency):

- find example code
- explore/read code
- debug issues
- determine the impact of changes

Sourcegraph code search helps developers perform these tasks more quickly and effectively by providing fast, advanced code search across multiple repositories. With Sourcegraph's code search, you can:

Sourcegraph provides fast, advanced code search across multiple repositories. With Sourcegraph's code search, you can:

- Use regular expressions and exact queries to perform full-text searches.
- Perform [language-aware structural search](#language-aware-structural-code-search) on code structure.
- Search any branch and commit, with no indexing required.
- Search [commit diffs](#commit-diff-search) and [commit messages](#commit-message-search) to see how code has changed.
- Narrow your search by repository and file pattern.
- Define saved [search scopes](#search-scopes) for easier searching.
- Curate [saved searches](#saved-searches) for yourself or your org.
- Set up notifications for code changes that match a query.
- View [language statistics](#statistics) for search results.

This document is for code search users. To get code search, [install Sourcegraph](../../admin/install/index.md).

---

## Details

### Data freshness

Searches scoped to specific repositories are always up-to-date. Sourcegraph automatically fetches repository contents with any user action specific to the repository and makes new commits and branches available for searching and browsing immediately.

Unscoped search results over large repository sets may trail latest default branch revisions by some interval of time. This interval is a function of the number of repositories and the computational resources devoted to search indexing.

### Max file size

By default, files larger than 1 MB are excluded from search results. Use the [search.largeFiles](../../admin/config/site_config.md#search-largeFiles) keyword to specify files to be indexed and searched regardless of size.

### Exclude files and directories

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

---

## Other tips

- When viewing a file or directory, press the `y` key to expand the URL to its canonical form (with the full 40-character Git commit SHA).
- To share a link to multi-line range in a file, click on the starting line number and shift-click on the ending line number (in the left-hand gutter).

---

## Sourcegraph Cloud

[Sourcegraph Cloud](https://sourcegraph.com/search) is a public instance of Sourcegraph that lets you search inside any open-source project on GitHub. For demo purposes, you'll be prompted to narrow your query if it would search across more than 50 repositories. To lift this limitation or to search your organization's internal code, [run your own Sourcegraph instance](../../admin/install/index.md).
