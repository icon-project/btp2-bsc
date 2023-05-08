ARG BASE_IMAGE
FROM ${BASE_IMAGE}
LABEL MAINTAINER="t_arch@iconloop.com"

RUN apk add --no-cache libstdc++

ARG VERSION
LABEL RELAY_VERSION="$VERSION"

# install
COPY dist/bin/relay /relay/bin/relay
ENV PATH $PATH:/relay/bin

WORKDIR /relay

# container configuration
VOLUME ["/relay/data"]

CMD relay start
