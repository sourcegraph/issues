
# Search query syntax

<!-- Search syntax styling overrides -->
<style>
.markdown-body tr td:nth-child(1) {
  min-width:175px;
}

.markdown-body tr td:nth-child(3) {
  min-width: 250px;
}
.markdown-body tr td:nth-child(3) code {
  word-break: break-all;
}

body.theme-dark img.toggle {
    filter: invert(100%);
}

img.toggle {
    width: 20px;
    height: 20px;
}

.toggle-container {
  border: 1px solid;
  border-radius: 3px;
  display: inline-flex;
  vertical-align: bottom;
}

</style>

This page describes search pattern syntax and keywords available for code search. See the complementary [language reference](language.md) for a visual breakdown. A typical search pattern describes content or filenames to find across all repositories. At the most basic level, a search pattern can simply be a word like `hello`. See our [search patterns](#search-pattern-syntax) documentation for detailed usage. Queries can also include keywords. For example, a typical search query will include a `repo:` keyword that filters search results for a specific repository. See our [keywords](#keywords-all-searches) documentation for more examples.

## Search pattern syntax

This section documents the available search pattern syntax and interpretation in Sourcegraph. A search pattern is _required_ to match file content. A search pattern is _optional_ and may be omitted when searching for [commits](#keywords-diff-and-commit-searches-only), [filenames](#filename-search), or [repository names](#repository-name-search).

### Literal search (default)

Literal search interprets search patterns literally to simplify searching for words or punctuation.

| Search pattern syntax                                                             | Description                                                                                                                                                                                                                           |
| ---                                                                               | ---                                                                                                                                                                                                                                   |
| [`foo bar`](https://sourcegraph.com/search?q=foo+bar&patternType=literal)         | Match the string `foo bar`. Matching is ordered: match `foo` followed by `bar`. Matching is case-_insensitive_ (toggle the <span class="toggle-container"><img class="toggle" src=../img/case.png alt="case"></span> button to change). |
| [`"foo bar"`](https://sourcegraph.com/search?q=%22foo+bar%22&patternType=literal) | Match the string `"foo bar"`. The quotes are matched literally.                                                                                                                                                                       |

### Regular expression search

Click the <span class="toggle-container"><img class="toggle" src=../img/regex.png alt="regular expression"></span> toggle to interpret search patterns as regexps. [RE2 syntax](https://golang.org/s/re2syntax) is supported. In general, special characters may be escaped with `\`. Here is a list of valid syntax and behavior:

| Search pattern syntax | Description |
| --- | --- |
| [`foo bar`](https://sourcegraph.com/search?q=foo+bar&patternType=regexp) | Search for the regexp `foo(.*?)bar`. Spaces between non-whitespace strings is converted to `.*?` to create a fuzzy search. Matching is case _insensitive_ (toggle the <span class="toggle-container"><img class="toggle" src=../img/case.png alt="case"></span> button to change). |
| [`foo\ bar`](https://sourcegraph.com/search?q=foo%5C+bar&patternType=regexp) or<br/>[`/foo bar/`](https://sourcegraph.com/search?q=/foo+bar/&patternType=regexp) | Search for the regexp `foo bar`. The `\` escapes the space and treats the space as part of the pattern. Using the delimiter syntax `/ ... /` avoids the need for escaping spaces. |
| [`foo\nbar`](https://sourcegraph.com/search?q=foo%5Cnbar&patternType=regexp) | Perform a multiline regexp search. `\n` is interpreted as a newline. |
| [`"foo bar"`](https://sourcegraph.com/search?q=%27foo+bar%27&patternType=regexp) | Match the _string literal_ `foo bar`. Quoting strings when regexp is active means patterns are interpreted [literally](#literal-search-default), except that special characters like `"` and `\` may be escaped, and whitespace escape sequences like `\n` are interpreted normally. |

### Structural search

Click the <span class="toggle-container"><img class="toggle" src=../img/brackets.png alt="square brackets"></span> toggle to activate structural search. Structural search is a way to match richer syntactic structures like multiline code blocks. See the dedicated [usage documentation](structural.md) for more details. Here is a  brief overview of valid syntax:

| Search pattern syntax | Description |
| --- | --- |
| [`New(ctx, ...)`](https://sourcegraph.com/search?q=repo:github.com/sourcegraph/sourcegraph++New%28ctx%2C+...%29+lang:go&patternType=structural) | Match call-like syntax with an identifier `New` having two or more arguments, and the first argument matches `ctx`. Make the search language-aware by adding a `lang:` [keyword](#keywords-all-searches). |

## Keywords (all searches)

The following keywords can be used on all searches (using [RE2 syntax](https://golang.org/s/re2syntax) any place a regex is accepted):

| Keyword | Description | Examples |
| --- | --- | --- |
| **repo:regexp-pattern** <br> **repo:regexp-pattern@rev** <br> **repo:regexp-pattern rev:rev**<br>_alias: r_  | Only include results from repositories whose path matches the regexp-pattern. A repository's path is a string such as _github.com/myteam/abc_ or _code.example.com/xyz_ that depends on your organization's repository host. If the regexp ends in [`@rev`](#repository-revisions), that revision is searched instead of the default branch (usually `master`).  `repo:regexp-pattern@rev` is equivalent to `repo:regexp-pattern rev:rev`.| [`repo:gorilla/mux testroute`](https://sourcegraph.com/search?q=repo:gorilla/mux+testroute) <br/> [`repo:^github\.com/sourcegraph/sourcegraph$@v3.14.0 mux`](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24%40v3.14.0+mux&patternType=literal) |
| **-repo:regexp-pattern** <br> _alias: -r_ | Exclude results from repositories whose path matches the regexp. | `repo:alice/ -repo:old-repo` |
|**rev:revision-pattern** <br> _alias: revision_| Search a revision instead of the default branch. `rev:` can only be used in conjunction with `repo:` and may not be used more than once. See our [revision syntax](#repository-revisions) documentation to learn more.| [`repo:sourcegraph/sourcegraph rev:v3.14.0 mux`](https://sourcegraph.com/search?q=repo:sourcegraph/sourcegraph+rev:v3.14.0+mux&patternType=literal) |
| **repogroup:group-name** <br> _alias: g_ | Only include results from the named group of repositories (defined by the server admin). Same as using a repo: keyword that matches all of the group's repositories. Use repo: unless you know that the group exists. | |
| **file:regexp-pattern** <br> _alias: f_ | Only include results in files whose full path matches the regexp. | [`file:\.js$ httptest`](https://sourcegraph.com/search?q=file:%5C.js%24+httptest) <br> [`file:internal/ httptest`](https://sourcegraph.com/search?q=file:internal/+httptest) |
| **-file:regexp-pattern** <br> _alias: -f_ | Exclude results from files whose full path matches the regexp. | [`file:\.js$ -file:test http`](https://sourcegraph.com/search?q=file:%5C.js%24+-file:test+http) |
| **content:"pattern"** | Set the search pattern with a dedicated parameter. Useful when searching literally for a string that may conflict with the [search pattern syntax](#search-pattern-syntax). | [`repo:sourcegraph content:"repo:sourcegraph"`](https://sourcegraph.com/search?q=repo:sourcegraph+content:"repo:sourcegraph"&patternType=literal) |
| **-content:"pattern"** | Exclude results from files whose content matches the pattern. See the [requirements and current support](#negated-content-search) for negated content search. | [`file:Dockerfile alpine -content:alpine:latest`](https://sourcegraph.com/search?q=file:Dockerfile+alpine+-content:alpine:latest&patternType=literal) |
| **lang:language-name** <br> _alias: l_ | Only include results from files in the specified programming language. | [`lang:typescript encoding`](https://sourcegraph.com/search?q=lang:typescript+encoding) |
| **-lang:language-name** <br> _alias: -l_ | Exclude results from files in the specified programming language. | [`-lang:typescript encoding`](https://sourcegraph.com/search?q=-lang:typescript+encoding) |
| **type:symbol** | Perform a symbol search. | [`type:symbol path`](https://sourcegraph.com/search?q=type:symbol+path)  ||
| **case:yes**  | Perform a case sensitive query. Without this, everything is matched case insensitively. | [`OPEN_FILE case:yes`](https://sourcegraph.com/search?q=OPEN_FILE+case:yes) |
| **fork:yes, fork:only** | Include results from repository forks or filter results to only repository forks. Results in repository forks are exluded by default. | [`fork:yes repo:sourcegraph`](https://sourcegraph.com/search?q=fork:yes+repo:sourcegraph) |
| **archived:yes, archived:only** | Include archived repositories or filter results to only archived repositories. Results in archived repositories are excluded by default. | [`repo:sourcegraph/ archived:only`](https://sourcegraph.com/search?q=repo:%5Egithub.com/sourcegraph/+archived:only) |
| **repohasfile:regexp-pattern** | Only include results from repositories that contain a matching file. This keyword is a pure filter, so it requires at least one other search term in the query.  Note: this filter currently only works on text matches and file path matches. | [`repohasfile:\.py file:Dockerfile pip`](https://sourcegraph.com/search?q=repohasfile:%5C.py+file:Dockerfile+pip+repo:/sourcegraph/) |
| **-repohasfile:regexp-pattern** | Exclude results from repositories that contain a matching file. This keyword is a pure filter, so it requires at least one other search term in the query. Note: this filter currently only works on text matches and file path matches. | [`-repohasfile:Dockerfile docker`](https://sourcegraph.com/search?q=-repohasfile:Dockerfile+docker) |
| **repohascommitafter:"string specifying time frame"** | (Experimental) Filter out stale repositories that don't contain commits past the specified time frame. | [`repohascommitafter:"last thursday"`](https://sourcegraph.com/search?q=error+repohascommitafter:%22last+thursday%22) <br> [`repohascommitafter:"june 25 2017"`](https://sourcegraph.com/search?q=error+repohascommitafter:%22june+25+2017%22) |
| **count:_N_**<br/> | Retrieve at least <em>N</em> results. By default, Sourcegraph stops searching early and returns if it finds a full page of results. This is desirable for most interactive searches. To wait for all results, or to see results beyond the first page, use the **count:** keyword with a larger <em>N</em>. This can also be used to get deterministic results and result ordering (whose order isn't dependent on the variable time it takes to perform the search). | [`count:1000 function`](https://sourcegraph.com/search?q=count:1000+repo:sourcegraph/sourcegraph$+function) |
| **timeout:_go-duration-value_**<br/> | Customizes the timeout for searches. The value of the parameter is a string that can be parsed by the [Go time package's `ParseDuration`](https://golang.org/pkg/time/#ParseDuration) (e.g. 10s, 100ms). By default, the timeout is set to 10 seconds, and the search will optimize for returning results as soon as possible. The timeout value cannot be set longer than 1 minute. When provided, the search is given the full timeout to complete. | [`repo:^github.com/sourcegraph timeout:15s func count:10000`](https://sourcegraph.com/search?q=repo:%5Egithub.com/sourcegraph/+timeout:15s+func+count:10000) |
| **patterntype:literal, patterntype:regexp, patterntype:structural**  | Configure your query to be interpreted literally, as a regular expression, or a [structural search pattern](structural.md). Note: this keyword is available as an accessibility option in addition to the visual toggles. | [`test. patternType:literal`](https://sourcegraph.com/search?q=test.+patternType:literal)<br/>[`(open\|close)file patternType:regexp`](https://sourcegraph.com/search?q=%28open%7Cclose%29file&patternType=regexp) |
| **visibility:any, visibility:public, visibility:private** | Filter results to only public or private repositories. The default is to include both private and public repositories. | [`type:repo visibility:public`](https://sourcegraph.com/search?q=type:repo+visibility:public) |
| **stable:yes** | Ensures a deterministic result order. Applies only to file contents. Limited to at max `count:5000` results. Note this field should be removed if you're using the pagination API, which already ensures deterministic results. | [`func stable:yes count:10`](https://sourcegraph.com/search?q=func+stable:yes+count:30&patternType=literal) |

Multiple or combined **repo:** and **file:** keywords are intersected. For example, `repo:foo repo:bar` limits your search to repositories whose path contains **both** _foo_ and _bar_ (such as _github.com/alice/foobar_). To include results from repositories whose path contains **either** _foo_ or _bar_, use `repo:foo|bar`.

## Operators

Use operators to create more expressive searches.

| Operator | Example |
| --- | --- |
| `and`, `AND` | [`conf.Get( and log15.Error(`](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+conf.Get%28+and+log15.Error%28&patternType=regexp), [`conf.Get( and log15.Error( and after`](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+conf.Get%28+and+log15.Error%28+and+after&patternType=regexp) |

Returns results for files containing matches on the left _and_ right side of the `and` (set intersection).

| Operator | Example |
| --- | --- |
| `or`, `OR` | [`conf.Get( or log15.Error(`](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+conf.Get%28+or+log15.Error%28&patternType=regexp), [<code>conf.Get( or log15.Error( or after   </code>](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+conf.Get%28+or+log15.Error%28+or+after&patternType=regexp)|

Returns file content matching either on the left or right side, or both (set union). The number of results reports the number of matches of both strings.

| Operator | Example |
| --- | --- |
| `not`, `NOT` | [`lang:go not file:main.go panic`](https://sourcegraph.com/search?q=lang:go+not+file:main.go+panic&patternType=literal), [`panic NOT ever`](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+panic+not+ever&patternType=literal)

`NOT` can be used in place of `-` to negate keywords, such as `file`, `content`, `lang`, `repohasfile`, and `repo`. For
search patterns, `NOT` excludes documents that contain the term after `NOT`. For readability, you can also include the
`AND` operator before a `NOT` (i.e. `panic NOT ever` is equivalent to `panic AND NOT ever`).


### Operator precedence and groups

Operators may be combined. `and`-expressions have higher precedence (bind tighter) than `or`-expressions so that `a and b or c and d` means `(a and b) or (c and d)`.

Expressions may be grouped with parentheses to change the default precedence and meaning. For example: `a and (b or c) and d`.

### Operator scope

Except for simple cases, search patterns bind tightest to scoped fields, like `file:main.c`. So, a combined query like
`file:main.c char c  or (int i and int j)` generally means `(file:main.c char c) or (int i and int j)`

Since we don't yet support search subexpressions with different scopes, the above will raise an alert. If the intent is to apply the `file` scope to the entire pattern, group it like so: `file:main.c (char c or (int i and int j))`

### Operator support

Operators are supported in regexp and structural search modes, but not literal search mode. How operators interpret search pattern syntax depends on kind of search (whether [regexp](#regexp-search) or [structural](#structural-search)). Operators currently only apply to searches for file content. Thus, expressions like `repo:npm/cli or repo:npm/npx` are not currently supported.

---

## Keywords (diff and commit searches only)

The following keywords are only used for **commit diff** and **commit message** searches, which show changes over time:

| Keyword  | Description | Examples |
| --- | --- | --- |
| **repo:regexp-pattern@rev** | Specifies which Git revisions to search for commits. See our [repository revisions](#repository-revisions) documentation to learn more about the revision syntax. | [`repo:vscode@*refs/heads/:^refs/heads/master type:diff task`](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/Microsoft/vscode%24%40*refs/heads/:%5Erefs/heads/master+type:diff+after:%221+month+ago%22+task#1) (unmerged commit diffs containing `task`) |
| **type:diff** <br> **type:commit**  | Specifies the type of search. By default, searches are executed on all code at a given point in time (a branch or a commit). Specify the `type:` if you want to search over changes to code or commit messages instead (diffs or commits).  | [`type:diff func`](https://sourcegraph.com/search?q=type:diff+func+repo:sourcegraph/sourcegraph$) <br> [`type:commit test`](https://sourcegraph.com/search?q=type:commit+test+repo:sourcegraph/sourcegraph$) |
| **author:name** | Only include results from diffs or commits authored by the user. Regexps are supported. Note that they match the whole author string of the form `Full Name <user@example.com>`, so to include only authors from a specific domain, use `author:example.com>$`.<br><br> You can also search by `committer:git-email`. _Note: there is a committer only when they are a different user than the author._ | [`type:diff author:nick`](https://sourcegraph.com/search?q=repo:sourcegraph/sourcegraph$+type:diff+author:nick) |
| **-author:name** | Exclude results from diffs or commits authored by the user. Regexps are supported. Note that they match the whole author string of the form `Full Name <user@example.com>`, so to exclude authors from a specific domain, use `author:example.com>$`.<br><br> You can also search by `committer:git-email`. _Note: there is a committer only when they are a different user than the author._ | [`type:diff author:nick`](https://sourcegraph.com/search?q=repo:sourcegraph/sourcegraph$+type:diff+author:nick) |
| **before:"string specifying time frame"** | Only include results from diffs or commits which have a commit date before the specified time frame | [`before:"last thursday"`](https://sourcegraph.com/search?q=repo:sourcegraph/sourcegraph$+type:diff+author:nick+before:%22last+thursday%22) <br> [`before:"november 1 2019"`](https://sourcegraph.com/search?q=repo:sourcegraph/sourcegraph$+type:diff+author:nick+before:%22november+1+2019%22) |
| **after:"string specifying time frame"**  | Only include results from diffs or commits which have a commit date after the specified time frame| [`after:"6 weeks ago"`](https://sourcegraph.com/search?q=repo:sourcegraph/sourcegraph$+type:diff+author:nick+after:%226+weeks+ago%22) <br> [`after:"november 1 2019"`](https://sourcegraph.com/search?q=repo:sourcegraph/sourcegraph$+type:diff+author:nick+after:%22november+1+2019%22) |
| **message:"any string"** | Only include results from diffs or commits which have commit messages containing the string | [`type:commit message:"testing"`](https://sourcegraph.com/search?q=type:commit+repo:sourcegraph/sourcegraph$+message:%22testing%22) <br> [`type:diff message:"testing"`](https://sourcegraph.com/search?q=type:diff+repo:sourcegraph/sourcegraph$+message:%22testing%22) |
| **-message:"any string"** | Exclude results from diffs or commits which have commit messages containing the string | [`type:commit message:"testing"`](https://sourcegraph.com/search?q=type:commit+repo:sourcegraph/sourcegraph$+message:%22testing%22) <br> [`type:diff message:"testing"`](https://sourcegraph.com/search?q=type:diff+repo:sourcegraph/sourcegraph$+message:%22testing%22) |

## Repository search

### Repository revisions

To search revisions other than the default branch, specify the revisions by either appending them to the
`repo:` filter  or by listing them separately with the `rev:` filter. This means:

`repo:github.com/myteam/abc@<revisions>`

is equivalent to

`repo:github.com/myteam/abc rev:<revisions>`.

 The `<revisions>` part refers to repository
 revisions (branches, commit hashes, and tags) and may take on the following forms:

(All examples apply to `@` as well as `rev:`)

- `@branch` - a branch name
- `@1735d48` - a commit hash
- `@3.15` - a tag

You can separate revisions by a colon to search multiple revisions at the same time, `@branch:1735d48:3.15`.

Per default, we match revisions to tags, branches, and commits. You can limit the search to branches or tags by adding
the prefix `refs/tags` or `refs/heads`. For example `@refs/tags/3.18` will search the commit tagged
with `3.18`, but not a branch called `3.18` and vice versa for `@refs/heads/3.18`.

**Glob patterns** allow you to search over a range of branches or tags. Prepend `*` to mark a revision
as glob pattern and add the glob-pattern after it like this `repo:<repo>@*<glob-pattern>`. For example:

 - [`@*refs/heads/*`](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/docker/machine%24%40*refs/heads/*+middleware&patternType=literal) - search across all branches
 - [`@*refs/tags/*`](https://sourcegraph.com/search?q=repo:github.com/docker/machine%24%40*refs/tags/*+server&patternType=literal) - search across all tags

We automatically add a trailing `/*` if it is missing from the glob pattern.

You can negate a glob pattern by prepending `*!`, for example:

- [`@*refs/heads/*:*!refs/heads/release* type:commit `](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/kubernetes/kubernetes%24%40*refs/heads/*:*%21refs/heads/release*+type:commit+&patternType=literal) - search commits on all branches except on those that start with "release"
- [`@*refs/tags/v3.*:*!refs/tags/v3.*-* context`](https://sourcegraph.com/search?q=repo:%5Egithub.com/sourcegraph/sourcegraph%24%40*refs/tags/v3.*:*%21refs/tags/v3.*-*+context&patternType=literal) - search all versions starting with `3.` except release candidates, alpha and beta versions.

#### Commit and Diff searches
Commit and diff searches act on sets of commits. A set is defined by a revision (branch, commit hash, or tag), and it
contains all commits reachable from that revision. A commit is reachable from another commit if it can be
reached by following the pointers to parent commits.

For commit and diff searches it is possible to exclude a set of commits by prepending a caret `^`. The caret acts as a set
difference. For example, `repo:github.com/myteam/abc@main:^3.15 type:commit` will show all commits in `main`
minus the commits reachable from the commit tagged with `3.15`.

### Repository names

A query with only `repo:` filters returns a list of repositories with matching names.

Example: [`repo:docker repo:registry`](https://sourcegraph.com/search?q=repo:docker+repo:registry)

## Filename search

A query with `type:path` restricts terms to matching filenames only (not file contents).

Example: [`type:path repo:/docker/ registry`](https://sourcegraph.com/search?q=type:path+repo:/docker/+registry)


## Negated content search

To exclude code or text matches with `not pattern` or `-content:pattern`, set the `"search.migrateParser": true` option in global settings. Negated content search is currently supported for literal and regexp queries.
