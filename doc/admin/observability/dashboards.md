# Grafana dashboards

When you click on **Site admin > Monitoring** you will see the Grafana Home Dashboard.

![Home Dashboard](https://sourcegraphstatic.com/GrafanaHomeDashboard.png)

> NOTE: We are working on changing the default Home Dashboard to show useful summary metrics panels.

Click on the grid symbol in the upper left side bar (marked by the red arrow in the image above) and choose "Manage" to
get to the list of available dashboards.

Here are some of the dashboards in more detail:

### Cluster-Internal Network Activity, Client POV

This dashboard has panels showing the request rate, error rate and request latency of requests made to gitserver and
repo-updater. This covers some of the internal network traffic as seen by frontend, searcher etc.

### Gitserver rev2

This dashboard shows the health of the gitserver(s). This includes available disk space, number of commands running concurrently
and how long each command takes. It also has stats on the set of repositories processed by a gitserver.

### Go processes

Panels for CPU, memory, open file descriptors, number of goroutines and garbage collector stats.

### repo-updater to external services

Shows traffic to configured code hosts. Dashboard has panels for Github, Gitlab and Bitbucket.

### Searcher

Dashboard showing request and caching stats from the searcher processes. 

### Symbols

Panels include stats for parsing and store fetching.

### Zoekt indexserver

Dashboard with indexing repositories and resolving revisions stats.

### LSIF

Panels related to the LSIF API server, bundle manager, and workers.
