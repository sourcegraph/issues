# Getting started

Commonly asked questions by new Sourcegraph users.

## What is Sourcegraph?

Sourcegraph is a code search and intelligence tool for developers. It lets you search and explore all of your organization's code on the web, with integrations into your existing tools.

## What does Sourcegraph do?

Sourcegraph's main features are:

- [Code search](overview.md#code-search): fast, up-to-date, and scalable, with regexp support on any branch or commit without an indexing delay (and diff search)
- [Code intelligence](overview.md#code-intelligence): jump-to-definition, find references, and other smart, IDE-like code browsing features on any branch, commit, or PR/code review
- [Campaigns](overview.md#campaigns): TODO
- [Integrations](overview.md#integrations) with code hosts, code review tools, editors, web browsers, etc.

Two options:

- [Sourcegraph Cloud](sourcegraph.com/search) for searching any public code from GitHub.com or GitLab.com
- [Sourcegraph Server](#): Easy and secure self-hosted installation for your private code (your code never touches our servers)

## What do I use Sourcegraph for?

Sourcegraph helps you:

- Find example code
- Explore/read code (including during a code review)
- Debug issues
- Locate a specific piece of code
- Determine the impact of changes

Sourcegraph makes it faster and easier to perform these tasks, for you and everyone else at your organization.

## Who should use Sourcegraph?

Sourcegraph would be most useful to developers who frequently search, read, or review code, ideally working with larger codebases or teams of 15 or more.

If these conditions do not apply, Sourcegraph may not be useful for you. (But you should start reading and reviewing more code!)

## Who else uses Sourcegraph?

In addition to the [companies listed on about.sourcegraph.com](https://about.sourcegraph.com), many large technology companies with 500-3,000+ engineers use Sourcegraph internally.

Also, both Facebook and Google provide an in-house Sourcegraph-like code search and intelligence tool to their employees. A [published research paper from Google](https://static.googleusercontent.com/media/research.google.com/en//pubs/archive/43835.pdf) and a [Google developer survey](https://docs.google.com/document/d/1LQxLk4E3lrb3fIsVKlANu_pUjnILteoWMMNiJQmqNVU/edit#heading=h.xxziwxixfqq3) showed that 98% of developers surveyed consider their Sourcegraph-like internal tool to be critical. Developers use it on average for 5.3 sessions each day. (Facebook's and Google's in-house tools are not available to other companies; use Sourcegraph instead.)

## How do I start using Sourcegraph?

1. [Install Sourcegraph](../admin/install/index.md) inside your organization on your internal code, if nobody else has yet
1. Install and configure the [web browser code host integrations](../integration/browser_extension.md) (recommended)
1. Start searching and browsing code on Sourcegraph by visiting the URL of your organization's internal Sourcegraph instance

You can also try [Sourcegraph.com](https://sourcegraph.com/search), which is a public instance of Sourcegraph for use on open-source code only.

## Why should my company use Sourcegraph?

- [Sourcegraph user features](../user/index.md)
- [Code search product comparisons](https://about.sourcegraph.com/workflow#other-tools)
- [How to run a Sourcegraph trial](how-to/run-a-sourcegraph-trial.md)
