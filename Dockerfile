FROM golang:1.16-buster AS builder
ARG BUILD_VERSION
ENV BUILD_VERSION=${BUILD_VERSION}
ADD . /ffsigner
WORKDIR /ffsigner
RUN make

FROM debian:buster-slim
WORKDIR /ffsigner
COPY --from=builder /ffsigner/firefly-signer /usr/bin/ffsigner

ENTRYPOINT [ "/usr/bin/ffsigner" ]
