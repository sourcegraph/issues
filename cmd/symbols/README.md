# symbols

Indexes symbols in repositories using [Ctags](https://github.com/universal-ctags/ctags). Similar in architecture to searcher, except over ctags output.

The ctags output is stored in SQLite files on disk (one per repository@commit). Ctags processing is lazy, so it will occur only when you first query the symbols service. Subsequent queries will use the cached on-disk SQLite DB.

It is used by [basic-code-intel](https://github.com/sourcegraph/sourcegraph-basic-code-intel) to provide the jump-to-definition feature.

It supports regex queries, with queries of the form `^foo$` optimized to perform an index lookup (basic-code-intel takes advantage of this).

(c) Copyright Sourcegraph 2012-2023.