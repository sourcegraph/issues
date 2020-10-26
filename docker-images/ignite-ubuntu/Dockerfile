FROM weaveworks/ignite-ubuntu:20.04-amd64@sha256:4f5f5ed56fae650ae122daa28a785192dda081be4f0b37dca2eb25ea57840500

# hadolint ignore=DL3008,DL3009
RUN set -ex && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    docker.io

RUN mkdir -p /etc/docker/ && echo '{"registry-mirrors": ["http://localhost:5000"] }' >/etc/docker/daemon.json

ARG SRC_CLI_VERSION

RUN set -ex && \
    curl -L -o src-cli.tar.gz "https://github.com/sourcegraph/src-cli/releases/download/${SRC_CLI_VERSION}/src-cli_${SRC_CLI_VERSION}_linux_amd64.tar.gz" && \
    tar -xvzf src-cli.tar.gz src && \
    mv src /usr/local/bin/src && \
    chmod +x /usr/local/bin/src && \
    rm -rf src-cli.tar.gz
