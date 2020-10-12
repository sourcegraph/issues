# Pings

Sourcegraph periodically sends a ping to Sourcegraph.com to help our product and customer teams. It sends only the high-level data below. It never sends code, repository names, usernames, or any other specific data. To learn more, go to the **Site admin > Pings** page on your instance. (The URL is `https://sourcegraph.example.com/site-admin/pings`.)

## Critical telemetry

Critical telemetry includes only the high-level data below required for billing, support, updates, and security notices.

- Randomly generated site identifier
- The email address of the initial site installer (or if deleted, the first active site admin), to know who to contact regarding sales, product updates, security updates, and policy updates
- Sourcegraph version string (e.g. "vX.X.X")
- Deployment type (single Docker image, Docker Compose, Kubernetes cluster, or pure Docker cluster)
- License key associated with your Sourcegraph subscription
- Aggregate count of current monthly users
- Total count of existing user accounts

## Other telemetry

By default, Sourcegraph also aggregates usage and performance metrics for some product features. No personal or specific information is ever included.

- Whether the instance is deployed on localhost (true/false)
- Which category of authentication provider is in use (built-in, OpenID Connect, an HTTP proxy, SAML, GitHub, GitLab)
- Which code hosts are in use (GitHub, Bitbucket Server, GitLab, Phabricator, Gitolite, AWS CodeCommit, Other)
- Whether new user signup is allowed (true/false)
- Whether a repository has ever been added (true/false)
- Whether a code search has ever been executed (true/false)
- Whether code intelligence has ever been used (true/false)
- Aggregate counts of current daily, weekly, and monthly users
- Aggregate counts of current daily, weekly, and monthly users, by:
  - Whether they are using code host integrations
  - Product area (site management, code search and navigation, code review, saved searches, diff searches)
  - Search modes used (interactive search, plain-text search)
  - Search filters used (e.g. "type:", "repo:", "file:", "lang:", etc.)
- Aggregate daily, weekly, and monthly latencies (in ms) of code intelligence events (e.g., hover tooltips) and search queries
- Aggregate daily, weekly, and monthly counts of:
  - Code intelligence events (e.g., hover tooltips) 
  - Searches using each search mode (interactive search, plain-text search)
  - Searches using each search filter (e.g. "type:", "repo:", "file:", "lang:", etc.)
<!-- depends-on-source: ~/cmd/frontend/internal/usagestats/campaigns.go -->
- Campaign usage data
  - Total count of created campaigns
  - Total count of changesets created by campaigns
  - Total count of changesets created by campaigns that have been merged
  - Total count of changesets manually added to a campaign
  - Total count of changesets manually added to a campaign that have been merged
- Aggregated counts of users created, deleted, retained, resurrected and churned within the month
- Saved searches usage data
  - Count of saved searches
  - Count of users using saved searches
  - Count of notifications triggered
  - Count of notifications clicked
  - Count of saved search views
- Aggregated repository statistics
  - Total size of git repositories stored in bytes
  - Total number of lines of code stored in text search index
