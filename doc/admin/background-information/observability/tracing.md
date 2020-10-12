# Tracing

## Prerequisites

### 1. Ensure Jaeger is running.

* **Single Docker container:** Jaeger will be integrated into the Sourcegraph single Docker container starting in 3.16.
* **Docker Compose:** Jaeger is deployed if you use the provided `docker-compose.yaml`. Access it at
  port 16686 on the Sourcegraph node. One way to do this is to add an Ingress rule exposing port
  16686 to public Internet traffic from your IP, then navigate to `http://${NODE_IP}:16686` in your
  browser.
* **Kubernetes:** Jaeger is already deployed, unless you explicitly removed it from the Sourcegraph
  manifest. Access it by running `kubectl port-forward svc/jaeger-query 16686` and going to
  `http://localhost:16686` in your browser.

The Jaeger UI should look something like this:

![Jaeger UI](https://user-images.githubusercontent.com/1646931/79700938-0586c600-824e-11ea-9c8c-a115df8b3a21.png)

### 2. Turn on sending traces to Jaeger from Sourcegraph:

1. Go to [site configuration](../config/site_config.md), add the following, and save:

   ```
   "observability.tracing": {
     "sampling": "selective"
   }
   ```
1. Go to Sourcegraph in your browser and do a search.
1. Open Chrome dev tools.
1. Append `&trace=1` to the end of the URL and hit `Enter`.
1. In the Chrome dev tools Network tab, find the `graphql?Search` request. Click it and click on the
   `Headers` tab. The value of the `x-trace` Response Header should be a trace ID, e.g.,
   `7edb43f744c42fbf`.

## Using Jaeger

In site configuration, you can configure the Jaeger client to use different sampling modes. There
are currently two modes:

* `"selective"` (recommend) will cause a trace to be recorded only when `trace=1` is present as a
  URL parameter.
* `"all"` will cause a trace to be recorded on every request.

`"selective"` is the recommended default, because collecting traces on all requests can be quite
memory- and network-intensive. If you have a large Sourcegraph instance (e.g,. more than 10k
repositories), turn this on with caution. You may need to increase the memory/CPU quota for the
Jaeger instance or [set a downsampling rate in Jaeger
itself](https://www.jaegertracing.io/docs/1.17/sampling/), and even then, the volume of network
traffic caused by Jaeger spans being sent to the collector may disrupt the performance of the
overall Sourcegraph instance.

### Jaeger debugging algorithm

Jaeger is a powerful debugging tool that can break down where time is spent over the lifecycle of a
request and help pinpoint the source of high latency or errors. We generally follow the following
algorithm to root-cause issues with Jaeger:

1. Reproduce a slower user request (e.g., a search query that takes too long or times out).
1. Add `trace=1` to the slow URL and reload the page, so that traces will be collected.
1. Open Chrome developer tools to the Network tab and find the corresponding GraphQL request that
   takes a long time. If there are multiple requests that take a long time, investigate them one by
   one.
1. In the Response Headers for the slow GraphQL request, find the `x-trace` header. It should
   contain a trace ID like `7edb43f744c42fbf`.
1. Go to the Jaeger UI and paste in the trace ID to the "Lookup by Trace ID" input in the top menu
   bar.
1. Explore the breakdown of the request tree in the Jaeger UI. Look for items near the leaves that
   take up a significant portion of the overall request time.
1. Report this information to Sourcegraph by screenshotting the relevant trace or by downloading the
   trace JSON.

## net/trace

Sourcegraph uses the [`net/trace`](https://pkg.go.dev/golang.org/x/net/trace) package in its backend
services. This provides simple tracing information within a single process. It can be used as an
alternative when Jaeger is not available or as a supplement to Jaeger.

Site admins can access `net/trace` information at https://sourcegraph.example.com/-/debug/. From
there, click **Requests** to view the traces for that service.
