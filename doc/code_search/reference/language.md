# Sourcegraph search query language

<style>

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

li.r {
    margin-top:10px !important;
    list-style:none !important;
}

.r {
    border: 0px !important;
    padding: 0px !important;
    margin: 0px !important;
    border-collapse: collapse !important;
    vertical-align: top !important;
    background-color: transparent !important;
}

th.r {
    text-align: left !important;
    padding: 3px !important;
}

td.r {
    text-align: left !important;
    vertical-align: top !important;
    border: 1px solid #aca899 !important;
    padding: 3px !important;
}


svg.railroad-diagram path {
    stroke-width: 2;
    stroke: var(--text-color);
    fill: rgba(0,0,0,0);
}

svg.railroad-diagram text {
    font: 14px var(--monospace-font-family);
    fill: var(--text-color);
    text-anchor: middle;
    white-space: pre;
}
svg.railroad-diagram a text {
    fill: var(--link-color);
}
svg.railroad-diagram a:hover text {
    text-decoration: underline;
}
svg.railroad-diagram text.diagram-text {
    font-size: 12px;
}
svg.railroad-diagram text.diagram-arrow {
    font-size: 16px;
}
svg.railroad-diagram text.label {
    text-anchor: start;
}
svg.railroad-diagram text.comment {
    font: italic 12px monospace;
}
svg.railroad-diagram g.non-terminal text {
    /*font-style: italic;*/
}
svg.railroad-diagram rect {
    stroke-width: 2;
    stroke: var(--text-color);
    fill: none;
}
svg.railroad-diagram rect.group-box {
    stroke: gray;
    stroke-dasharray: 10 5;
}
svg.railroad-diagram path.diagram-text {
    stroke-width: 3;
    stroke: var(--text-color);
    cursor: help;
}
svg.railroad-diagram g.diagram-text:hover path.diagram-text {
    fill: #eee;
}

</style>

This page provides a visual breakdown of our Search Query Language and a handful
of examples to get you started. It is complementary to our [syntax reference](../reference/queries.md) and illustrates syntax using railroad diagrams instead of
tables.

**How to read railroad diagrams.** Follow the lines in these railroad diagrams to see
how pieces of syntax combine. When a line splits it means there are multiple options
available. When it is possible to repeat a previous syntax, the split line will loop back
on itself like this:

<script>
ComplexDiagram(
    OneOrMore(
    Terminal("repeatable"))).addTo();
</script>

## Basic query

<script>
ComplexDiagram(
    OneOrMore(
        Choice(0,
            Terminal("search pattern", {href: "#search-pattern"}),
            Terminal("parameter", {href: "#parameter"})))).addTo();
</script>

At a basic level, a query consists of [search patterns](#search-pattern) and [parameters](#parameter). Typical queries contain one or more space-separated search patterns that describe what to search, and parameters refine searches by filtering results or changing search behavior.

**Example:** [`repo:github.com/sourcegraph/sourcegraph file:schema.graphql The result` ↗](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+file:schema.graphql+The+result&patternType=literal)

## Expression


<script>
ComplexDiagram(
    Terminal("basic query", {href: "#basic-query"}),
    ZeroOrMore(
        Sequence(
            Choice(0,
                Terminal("AND"),
                Terminal("OR")),
            Terminal("basic query", {href: "#basic-query"})),
        null,
        'skip')).addTo();
</script>


Build query expressions by combining [basic queries](#basic-query) and operators like `AND` or `OR`.
Group expressions with parentheses to build more complex expressions. If there are no balanced parentheses, `AND` operators bind tighter, so `foo or bar and baz` means `foo or (bar and baz)`. You may also use lowercase `and` or `or`.

**Example:** [`repo:github.com/sourcegraph/sourcegraph rtr AND newRouter` ↗](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+rtr+AND+newRouter&patternType=literal)


## Search pattern

<script>
ComplexDiagram(
    Choice(0,
        Terminal("string", {href: "#string"}),
        Terminal("quoted string", {href: "#quoted-string"}))).addTo();
</script>

A pattern to search. By default the pattern is searched literally. The kind of search may be toggled to change how a pattern matches:
<ul class="r">
    <li class="r"><span class="toggle-container"><img class="toggle" src="../img/regex.png"></span> Perform a [regular expression search](queries.md#regular-expression-search). We support [RE2 syntax](https://golang.org/s/re2syntax). Quoting patterns performs a literal search.<br>
    <strong>Example:</strong><a href="https://sourcegraph.com/search?q=foo+bar&patternType=regexp"> <code>foo.*bar.*baz</code> ↗</a> <a href="https://sourcegraph.com/search?q=%22foo+bar%22&patternType=regexp"><code>"foo bar"</code> ↗</a></li>
    <li class="r"><span class="toggle-container"><img class="toggle" src="../img/brackets.png"></span> Perform a structural search. See our [dedicated documentation](queries.md#structural-search) to learn more about structural sexarch. <br><strong>Example:</strong> <a href="https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+fmt.Sprintf%28%22:%5Bformat%5D%22%2C+:%5Bargs%5D%29&patternType=structural"><code>fmt.Sprintf(":[format]", :[args])</code> ↗</a></li>
</ul>


## Parameter

<script>
ComplexDiagram(
    Choice(0,
        Terminal("repo", {href: "#repo"}),
        Terminal("file", {href: "#file"}),
        Terminal("content", {href: "#content"}),
        Terminal("select", {href: "#select"}),
        Terminal("language", {href: "#language"}),
        Terminal("type", {href: "#type"}),
        Terminal("case", {href: "#case"}),
        Terminal("fork", {href: "#fork"}),
        Terminal("archived", {href: "#archived"}),
        Terminal("repogroup", {href: "#repogroup"}),
        Terminal("repohasfile", {href: "#repo-has-file"}),
        Terminal("repohascommitafter", {href: "#repo-has-commit-after"}),
        Terminal("count", {href: "#count"}),
        Terminal("timeout", {href: "#timeout"}),
        Terminal("visibility", {href: "#visibility"}),
        Terminal("patterntype", {href: "#pattern-type"}))).addTo();
</script>

Search parameters allow you to filter search results or modify search behavior.

### Repo

<script>
ComplexDiagram(
    Choice(0,
        Skip(),
        Terminal("-"),
        Sequence(
            Terminal("NOT"),
            Terminal("space", {href: "#whitespace"}))),
    Choice(0,
        Terminal("repo:"),
        Terminal("r:")),
    Choice(0,
        Terminal("regex", {href: "#regular-expression"}),
        Terminal("built-in", {href: "#built-in-repo-predicate"})),
    Choice(0,
        Skip(),
        Sequence(
            Terminal("@"),
            Terminal("revision", {href: "#revision"})),
        Sequence(
            Terminal("space", {href: "#whitespace"}),
            Terminal("rev:"),
            Terminal("revision", {href: "#revision"})))).addTo();
</script>

Search repositories that match the regular expression. A `-` before `repo`
excludes the repository. By default the repository will be searched at the
`HEAD` commit of the default branch. You can optionally change the
[revision](#revision).

**Example:** [`repo:gorilla/mux testroute` ↗](https://sourcegraph.com/search?q=repo:gorilla/mux+testroute&patternType=regexp) [`-repo:gorilla/mux testroute` ↗](https://sourcegraph.com/search?q=-repo:gorilla/mux+testroute&patternType=regexp)

### Revision

<script>
ComplexDiagram(
    OneOrMore(
        Choice(0,
            Terminal("branch name"),
            Terminal("commit hash"),
            Terminal("git tag")),
            Terminal(":"))).addTo();
</script>


Search a repository at a given revision. For example, a branch name, commit hash, or git tag.

**Example:** [`repo:^github\.com/gorilla/mux$@948bec34 testroute` ↗](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/gorilla/mux%24%40948bec34+testroute&patternType=literal) or [`repo:^github\.com/gorilla/mux$ rev:v1.8.0 testroute` ↗](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/gorilla/mux+rev:v1.8.0+testroute&patternType=literal)

You can search multiple revisions by separating the revisions with `:`. Specify `HEAD` for the default branch.

**Example:** [`repo:^github\.com/gorilla/mux$@v1.7.4:v1.4.0 testing.T` ↗](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/gorilla/mux%24%40v1.7.4:v1.4.0+testing.T&patternType=literal) or [`repo:^github\.com/gorilla/mux$ rev:v1.7.4:v1.4.0 testing.T` ↗](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/gorilla/mux%24+rev:v1.7.4:v1.4.0+testing.T&patternType=literal)

### File

<script>
ComplexDiagram(
    Choice(0,
        Skip(),
        Terminal("-"),
        Sequence(
            Terminal("NOT"),
            Terminal("space", {href: "#whitespace"}))),
    Choice(0,
        Terminal("file:"),
        Terminal("f:")),
        Terminal("regular expression", {href: "#regular-expression"})).addTo();
</script>

Search files whose full path matches the regular expression. A `-` before `file`
excludes the file from being searched.

**Example:** [`file:\.js$ httptest` ↗](https://sourcegraph.com/search?q=file:%5C.js%24+httptest&patternType=regexp) [`file:\.js$ -file:test http` ↗](https://sourcegraph.com/search?q=file:%5C.js%24+-file:test+http&patternType=regexp)

### Language

<script>
ComplexDiagram(
    Choice(0,
        Terminal("language"),
        Terminal("lang"),
        Terminal("l"))).addTo();
</script>

Only search files in the specified programming language, like `typescript` or
`python`.

**Example:** [`lang:typescript encoding` ↗](https://sourcegraph.com/search?q=lang:typescript+encoding&patternType=regexp)

### Content


<script>
ComplexDiagram(
    Choice(0,
        Skip(),
        Terminal("-"),
        Sequence(
            Terminal("NOT"),
            Terminal("space", {href: "#whitespace"}))),
    Terminal("content:"),
    Terminal("quoted string", {href: "#quoted-string"})).addTo();
</script>

Set the search pattern to search using a dedicated parameter. Useful, for
example, when searching literally for a string like `repo:my-repo` that may
conflict with the syntax of parameters in this Sourcegraph language.

**Example:** [`repo:sourcegraph content:"repo:sourcegraph"` ↗](https://sourcegraph.com/search?q=repo:sourcegraph+content:%22repo:sourcegraph%22&patternType=literal)

### Select

<script>
ComplexDiagram(
    Terminal("select:"),
    Choice(0,
        Terminal("repo"),
        Sequence(
            Terminal("file"),
            Optional(
                Sequence(
                    Terminal("."),
                    Terminal("file kind", {href: "#file-kind"})),
                'skip')),
        Terminal("content"),
        Sequence(
            Terminal("symbol"),
            Optional(
                Sequence(
                    Terminal("."),
                    Terminal("symbol kind", {href: "#symbol-kind"})),
                'skip')),
        Sequence(
            Terminal("commit.diff"),
            Terminal("."),
            Terminal("modified lines", {href: "#modified-lines"})))).addTo();
</script>

Selects the specified result type from the set of search results. If a query produces results that aren't of the
selected type, the results will be converted to the selected type.

For example, the query `file:package.json lodash` will return content matches for `lodash` in `package.json` files.
If `select:repo` is added, the repository those matches belong to is pulled out and it now only returns
_repositories_ that contain `package.json` files that contain the term `lodash`. All selected results are deduplicated,
so if there are multiple content matches in a repository, `select:repo` will still only return unique results.

A query like `type:commit example select:symbol` will return no results because commits have no associated symbol
and cannot be converted to that type.

**Example:**
[`fmt.Errorf select:repo` ↗](https://sourcegraph.com/search?q=fmt.Errorf+select:repo&patternType=literal)
[`zoektSearch select:file` ↗](https://sourcegraph.com/search?q=zoektSearch+select:file&patternType=literal)

#### Symbol kind

<script>
ComplexDiagram(
    Choice(0,
        Terminal("file"),
        Terminal("module"),
        Terminal("namespace"),
        Terminal("package"),
        Terminal("class"),
        Terminal("method"),
        Terminal("property"),
        Terminal("field"),
        Terminal("constructor"),
        Terminal("enum"),
        Terminal("interface"),
        Terminal("function"),
        Terminal("variable"),
        Terminal("constant"),
        Terminal("string"),
        Terminal("number"),
        Terminal("boolean"),
        Terminal("array"),
        Terminal("object"),
        Terminal("key"),
        Terminal("null"),
        Terminal("enum-member"),
        Terminal("struct"),
        Terminal("event"),
        Terminal("operator"),
        Terminal("type-parameter"))).addTo();
</script>

Select a specific kind of symbol. For example `type:symbol select:symbol.function zoektSearch` will only return functions that contain the
literal `zoektSearch`.

**Example:**
[`type:symbol zoektSearch select:symbol.function` ↗](https://sourcegraph.com/search?q=type:symbol+zoektSearch+select:symbol.function&patternType=literal)

#### Modified lines

<script>
ComplexDiagram(
    Choice(0,
        Terminal("added"),
        Terminal("removed"))).addTo();
</script>

When searching commit diffs, select only diffs where the pattern matches on
`added` (respectively, `removed`) lines. For example, search for recent commits
that removed `TODO`s in your code.

<small>- Note: if _any_ line exists that satisfies the condition, the entire diff is included in the result set.</small><br>
<small>- Note: `type:diff` must be specified in the query.</small>

**Example:**

[`repo:^github\.com/sourcegraph/sourcegraph$ type:diff TODO select:commit.diff.removed` ↗](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+type:diff+TODO+select:commit.diff.removed+&patternType=literal)

#### File kind

<script>
ComplexDiagram(
    Choice(0,
        Terminal("directory"),
        Terminal("path"))).addTo();
</script>

Select only directory paths of file results with `select:file.directory`. This is useful for discovering the directory paths that specify a `package.json` file, for example.
`select:file.path` returns the full path for the file and is equivalent to `select:file`. It exists as a fully-qualified alternative.

**Example:** [`file:package\.json select:file.directory` ↗](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+file:package%5C.json+select:file.directory&patternType=literal)

### Type

<script>
ComplexDiagram(
    Terminal("type:"),
    Choice(0,
        Terminal("symbol"),
        Terminal("repo"),
        Terminal("path"),
        Terminal("file"),
        Sequence(
            Choice(0,
            Terminal("commit"),
            Terminal("diff")),
            Terminal("commit parameter", {href: "#commit-parameter"})))).addTo();
</script>

Set whether the search pattern should perform a search of a certain type.
Notable search types are symbol, commit, and diff searches.

**Example:** [`type:symbol path` ↗](https://sourcegraph.com/search?q=type:symbol+path) [`type:commit author:nick` ↗](https://sourcegraph.com/search?q=repo:sourcegraph/sourcegraph%24+type:commit+author:nick&patternType=regexp)

### Case

<script>
ComplexDiagram(
    Terminal("case:"),
    Choice(0,
        Terminal("yes"),
        Terminal("no"))).addTo();
</script>


Set whether the search pattern should be treated case-sensitively. This is
synonymous with the <span class="toggle-container"><img class="toggle" src=../img/case.png></span> toggle button.

**Example:** [`OPEN_FILE case:yes` ↗](https://sourcegraph.com/search?q=OPEN_FILE+case:yes)


### Fork

<script>
ComplexDiagram(
    Terminal("fork:"),
    Choice(0,
        Terminal("yes"),
        Terminal("no"),
        Terminal("only"))).addTo();
</script>

Set to `yes` if repository forks should be included or `only` if only forks
should be searched. Repository forks are excluded by default.

**Example:** [`fork:yes repo:sourcegraph` ↗](https://sourcegraph.com/search?q=fork:yes+repo:sourcegraph&patternType=regexp)

### Archived

<script>
ComplexDiagram(
    Terminal("archived:"),
    Choice(0,
        Terminal("yes"),
        Terminal("no"),
        Terminal("only"))).addTo();
</script>

Set to `yes` if archived repositories should be included or `only` if only
archives should be searched. Archived repositories are excluded by default.

**Example:** [`archived:only repo:sourcegraph` ↗](https://sourcegraph.com/search?q=archived:only+repo:sourcegraph&patternType=regexp)

### Repo group

<script>
ComplexDiagram(
    Choice(0,
        Terminal("repogroup:"),
        Terminal("g:")),
        Terminal("string")).addTo()
</script>

Only include results from the named group of repositories (defined by the server
admin). Same as using [repo](#repo) that matches all of the group’s
repositories. Use [repo](#repo) unless you know that the group
exists.

**Example:** [`repogroup:go-gh-100 helm` ↗](https://sourcegraph.com/search?q=repogroup:go-gh-100+helm&patternType=literal)  – searches the top 100 Go repositories on GitHub, ranked by stars.

### Repo has file

<script>
ComplexDiagram(
    Choice(0,
        Skip(),
        Terminal("-"),
        Sequence(
            Terminal("NOT"),
            Terminal("space", {href: "#whitespace"}))),
    Terminal("repohasfile:"),
    Terminal("regular expression", {href: "#regular-expression"})).addTo();
</script>

_Deprecated. Prefer [Repo contains file](#repo-contains-file)._ Only include results from repositories that contain a matching file. This
keyword is a pure filter, so it requires at least one other search term in the
query. Note: this filter currently only works on text matches and file path
matches.

**Example:** [`repohasfile:\.py file:Dockerfile$ pip` ↗](https://sourcegraph.com/search?q=repohasfile:%5C.py+file:Dockerfile%24+pip+repo:sourcegraph+&patternType=regexp)

### Repo has commit after

<script>
ComplexDiagram(
    Terminal("repohascommitafter:"),
    Terminal("quoted string", {href: "#quoted-string"})).addTo();
</script>

_Deprecated. Prefer [Repo contains commit after](#repo-contains-commit-after)._ Filter out stale repositories that don’t
contain commits past the specified time frame. This parameter is experimental.

**Example:** [`repo:github\.com/sourcegraph repohascommitafter:"1 week ago"` ↗](https://sourcegraph.com/search?q=context:global+repo:github%5C.com/sourcegraph+repohascommitafter:%221+week+ago%22&patternType=literal)

### Count

<script>
ComplexDiagram(
    Terminal("count:"),
    Choice(0,
        Terminal("number"),
        Terminal("all"))).addTo();
</script>

Retrieve N results. By default, Sourcegraph stops searching early and
returns if it finds a full page of results. This is desirable for most
interactive searches. To wait for all results, use **count:all**.

**Example:** [`count:1000 function` ↗](https://sourcegraph.com/search?q=count:1000+repo:sourcegraph/sourcegraph%24+function&patternType=regexp)
[`count:all err`↗](https://sourcegraph.com/search?q=repo:github.com/sourcegraph/sourcegraph+err+count:all&patternType=literal)

### Timeout

<script>
ComplexDiagram(
    Terminal("timeout:"),
    Terminal("time value")).addTo();
</script>


Set a search timeout. The time value is a string like 10s or 100ms, which is
parsed by the Go time
package's [ParseDuration](https://golang.org/pkg/time/#ParseDuration).
By default the timeout is set to 10 seconds, and the search will optimize for
returning results as soon as possible. The timeout value cannot be set longer
than 1 minute.

**Example:** [`timeout:15s count:10000 func` ↗](https://sourcegraph.com/search?q=repo:%5Egithub.com/sourcegraph/+timeout:15s+func+count:10000)  – sets a longer timeout for a search that contains _a lot_ of results.

### Visibility

<script>
ComplexDiagram(
    Terminal("visibility:"),
    Choice(0,
        Terminal("any"),
        Terminal("public"),
        Terminal("private"))).addTo();
</script>

Filter results to only public or private repositories. The default is to include
both private and public repositories.

**Example:** [`type:repo visibility:public` ↗](https://sourcegraph.com/search?q=type:repo+visibility:public&patternType=regexp)

### Pattern type

<script>
ComplexDiagram(
    Terminal("patterntype:"),
    Choice(0,
        Terminal("literal"),
        Terminal("regexp"),
        Terminal("structural"))).addTo();
</script>


Set whether the pattern should run a literal search, regular expression search,
or a structural search pattern. This parameter is available as a command-line and
accessibility option, and synonymous with the visual [search pattern](#search-pattern) toggles.
in [search pattern](#search-pattern).

## Built-in repo predicate

<script>
ComplexDiagram(
    Choice(0,
        Terminal("contains.content(...)", {href: "#repo-contains-content"}),
        Terminal("contains.file(...)", {href: "#repo-contains-file"}),
        Terminal("contains(...)", {href: "#repo-contains-file-and-content"}),
        Terminal("contains.commit.after(...)", {href: "#repo-contains-commit-after"}))).addTo();
</script>

### Repo contains file

<script>
ComplexDiagram(
    Terminal("contains.file"),
    Terminal("("),
    Terminal("regexp", {href: "#regexp"}),
    Terminal(")")).addTo();
</script>

Search only inside repositories that contain a file path matching the regular expression.

**Example:** [`repo:contains.file(README)` ↗](https://sourcegraph.com/search?q=repo:github%5C.com/sourcegraph/.*+repo:contains.file%28README%29&patternType=literal)

### Repo contains content

<script>
ComplexDiagram(
    Terminal("contains.content"),
    Terminal("("),
    Terminal("regexp", {href: "#regular-expression"}),
    Terminal(")")).addTo();
</script>

Search only inside repositories that contain file content matching the regular expression.

**Example:** [`repo:contains.content(TODO)` ↗](https://sourcegraph.com/search?q=repo:github%5C.com/sourcegraph/.*+repo:contains.content%28TODO%29&patternType=literal)

### Repo contains file and content

<script>
ComplexDiagram(
    Terminal("contains"),
    Terminal("("),
    Stack(
        Sequence(Terminal("file:"), Terminal("regexp", {href: "#regular-expression"}), Terminal("space", {href: "#whitespace"})),
        Sequence(Terminal("content:"),Terminal("regexp", {href: "#regular-expression"}))),
    Terminal(")")).addTo();
</script>

Search only inside repositories that contain a file matching the `file:` with `content:` filters.

**Example:** [`repo:contains(file:CHANGELOG content:fix)` ↗](https://sourcegraph.com/search?q=repo:github%5C.com/sourcegraph/.*+repo:contains%28file:CHANGELOG+content:fix%29&patternType=literal)

### Repo contains commit after

<script>
ComplexDiagram(
    Terminal("contains.commit.after"),
    Terminal("("),
    Terminal("string", {href: "#string"}),
    Terminal(")")).addTo();
</script>

Search only inside repositories that contain a a commit after some specified
time. See [git date formats](https://github.com/git/git/blob/master/Documentation/date-formats.txt)
for accepted formats. Use this to filter out stale repositories that don’t contain
commits past the specified time frame. This parameter is experimental.

**Example:** [`repo:contains.commit.after(1 month ago)` ↗](https://sourcegraph.com/search?q=repo:.*sourcegraph.*+repo:contains.commit.after%281+month+ago%29&patternType=literal)

## Built-in file predicate

<script>
ComplexDiagram(
    Choice(0,
        Terminal("contains.content(...)", {href: "#file-contains-content"}),
        Terminal("contains(...)", {href: "#file-contains-content"}))).addTo();
</script>

### File contains content

<script>
ComplexDiagram(
    Terminal("contains"),
	Optional(Terminal(".content")),
    Terminal("("),
    Terminal("regexp", {href: "#regexp"}),
    Terminal(")")).addTo();
</script>

Search only inside files that contain content matching the provided regexp pattern.

**Example:** [`file:contains(github\.com/sourcegraph/sourcegraph)` ↗](https://sourcegraph.com/search?q=repo:github%5C.com/sourcegraph/.*+repo:contains.file%28README%29&patternType=literal)

## Regular expression

<script>
ComplexDiagram(
    Choice(0,
        Terminal("string", {href: "#string"}),
        Terminal("quoted string", {href: "#quoted-string"}))).addTo();
</script>

A string that is interpreted as a <a href="https://golang.org/s/re2syntax">RE2</a> regular expression.

## String

<script>
ComplexDiagram(
    Terminal("string")).addTo();
</script>

An unquoted string is any contiguous sequence of characters not containing whitespace.

## Quoted string

<script>
ComplexDiagram(
    Choice(0,
        Terminal('"any string"'),
        Terminal("'any string'"))).addTo();
</script>

Any string, including whitespace, may be quoted with single `'` or double `"`
quotes. Quotes can be escaped with `\`. Literal `\` characters will need to be escaped, e.g., `\\`.

## Commit parameter

<script>
ComplexDiagram(
    OneOrMore(
        Choice(0,
            Terminal("author", {href: "#author"}),
            Terminal("before", {href: "#before"}),
            Terminal("after", {href: "#after"}),
            Terminal("message", {href: "#message"})))).addTo();
</script>

Set parameters that apply only to commit and diff searches.

### Author

<script>
ComplexDiagram(
    Terminal("author:"),
    Terminal("regular expression", {href: "#regular-expression"})).addTo();
</script>

Include commits or diffs that are authored by the user.

### Before

<script>
ComplexDiagram(
    Choice(0,
        Terminal("before:"),
        Terminal("until:")),
    Terminal("quoted string", {href: "#quoted-string"})).addTo();
</script>

Include results which have a commit date before the specified time frame.

**Example:** [`before:"last thursday"` ↗](https://sourcegraph.com/search?q=repo:sourcegraph/sourcegraph%24+type:diff+author:nick+before:%22last+thursday%22&patternType=regexp) [`before:"november 1 2019"` ↗](https://sourcegraph.com/search?q=repo:sourcegraph/sourcegraph$+type:diff+author:nick+before:%22november+1+2019%22)

### After

<script>
ComplexDiagram(
    Choice(0,
        Terminal("after:"),
        Terminal("since:")),
    Terminal("quoted string", {href: "#quoted-string"})).addTo();
</script>

Include results which have a commit date before the specified time frame.

**Example:** [`after:"6 weeks ago"` ↗](https://sourcegraph.com/search?q=repo:sourcegraph/sourcegraph$+type:diff+author:nick+after:%226+weeks+ago%22) [`after:"november 1 2019"` ↗](https://sourcegraph.com/search?q=repo:sourcegraph/sourcegraph$+type:diff+author:nick+after:%22november+1+2019%22)

### Message

<script>
ComplexDiagram(
    Choice(0,
        Terminal("message:"),
        Terminal("msg:"),
        Terminal("m:")),
    Terminal("quoted string", {href: "#quoted-string"})).addTo();
</script>

Include results which have commit messages containing the string.

**Example:** [`type:commit message:"testing"` ↗](https://sourcegraph.com/search?q=type:commit+message:%22testing%22+repo:sourcegraph/sourcegraph%24+&patternType=regexp)

## Whitespace

<script>
ComplexDiagram(
    OneOrMore(
        Terminal("space"))).addTo();
</script>
