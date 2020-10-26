FROM gcr.io/cadvisor/cadvisor:v0.37.0@sha256:aa9694377d3be56d53b81b0aeecd5880c0832100923e3ed50a8229ef3e1704b1
LABEL com.sourcegraph.cadvisor.version=v0.37.0

ARG COMMIT_SHA="unknown"
ARG DATE="unknown"
ARG VERSION="unknown"

LABEL org.opencontainers.image.revision=${COMMIT_SHA}
LABEL org.opencontainers.image.created=${DATE}
LABEL org.opencontainers.image.version=${VERSION}
LABEL org.opencontainers.image.url=https://sourcegraph.com/
LABEL org.opencontainers.image.source=https://github.com/sourcegraph/sourcegraph/
LABEL org.opencontainers.image.documentation=https://docs.sourcegraph.com/

# Reflects cAdvisor Dockerfile at https://github.com/google/cadvisor/blob/v0.37.0/deploy/Dockerfile
# alongside additional Sourcegraph defaults.
ENTRYPOINT ["/usr/bin/cadvisor", "-logtostderr", \
    # sourcegraph cAdvisor custom port
    "-port=48080", \
    # only enable certain metrics, based on kubelet master
    "-disable_metrics=percpu,hugetlb,sched,tcp,udp,advtcp", \
    # other kubelet defaults
    # see https://sourcegraph.com/github.com/google/cadvisor@v0.37.0/-/blob/deploy/kubernetes/overlays/examples/cadvisor-args.yaml
    "-housekeeping_interval=10s", \
    "-max_housekeeping_interval=15s", \
    "-event_storage_event_limit=default=0", \
    "-event_storage_age_limit=default=0"]
