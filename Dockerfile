FROM golang:1.17-buster AS builder
ARG BUILD_VERSION
ENV BUILD_VERSION=${BUILD_VERSION}
ADD . /ffsigner
WORKDIR /ffsigner
RUN make

FROM debian:buster-slim
WORKDIR /ffsigner
RUN apt update -y \
 && apt install -y curl jq \
 && rm -rf /var/lib/apt/lists/*
COPY --from=builder /ffsigner/firefly-signer /usr/bin/ffsigner

ENTRYPOINT [ "/usr/bin/ffsigner" ]
