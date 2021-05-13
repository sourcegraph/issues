# Organizing Batch Changes with repository labels

Tracking the progress of large scale batch changes requires making sense of which repositories get their changeset merged and which require attention. In large batch changes with hundreds or thousands of changesets, users often find it useful to sort and filter by team, organisation or project.

To do so, you can associate each repository with one or more label, and then use it to filter changesets by label and  merge status. Because labels can be of any format, this allows you to customise this workflow to your organisation.

Common use cases include:

- Adding `project/project-id` labels to repositories, and then track progress of a batch change by project
- Adding a `team/team-name` label, and then track progress of a batch change by team or organisation
- Adding a `build-tool/version` label, and then track progress of a batch change by build tool version<Demo video>

< demo video >
## Adding labels

### Manually
You can add labels manually from the repository page.
<img src="https://sourcegraphstatic.com/batch-change-labels-add.png" class="screenshot">

### With a query
You can also create labels automatically, for example by loading metadata from an external service, with a GraphQL query:
```
TODO: query
```

## Removing labels
You can remove labels from the GUI, or with a query:
```
TODO: query
```


## Analysing progress of large batch changes by tag

You can filter by tag (and status) from the batch change page.
<img src="https://sourcegraphstatic.com/batch-change-labels-sort.png" class="screenshot">

You can also filter by tag in the burndown chart.

## Bulk Actions
Once you have filtered the changesets that require attention, you can use [bulk actions]() to either comment on, close, or merge a group of changesets


## Limitations and future work

There are known limitations to the first version of this feature:

- It's not possible to create a batch change based on a label filter in the `on.repositoriesMatchingQuery` statement
