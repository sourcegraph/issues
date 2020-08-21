FROM alpine:3.12@sha256:185518070891758909c9f839cf4ca393ee977ac378609f700f60a771a2dfe321

# hadolint ignore=DL3018
RUN apk add --no-cache bash

COPY /rootfs /

RUN /alpine-build-git.sh

