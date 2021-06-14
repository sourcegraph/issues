# How to find repositories with specific files
This document will take you through how to find a number of repositories with specific files or file formats.

For example, perhaps you would like to know how many repositories have `yaml` or `gradle` files. 
## Prerequisites
A running instance of Sourcegraph.
## Steps
1. On the search bar, run `file:<$name_of_file>(.<$file_extension>) select:repo `
2. If you'd like to know how many repos have this file, add the `count:all` filter.

## Further resources

- [Search subexpressions](https://docs.sourcegraph.com/code_search/tutorials/search_subexpressions)
