FROM golang:1.18-alpine AS builder

RUN set -xe && \
    apk upgrade --update-cache --available && \
    apk add  alpine-sdk openssh && \
    rm -rf /var/cache/apk/*


WORKDIR /builder
COPY go.mod .
COPY go.sum .

ENV GO111MODULE=on
RUN go mod download

WORKDIR /usr/local/go/src/
COPY . .
RUN go mod vendor
RUN make build

FROM alpine:latest
RUN set -xe && \
    apk upgrade --update-cache --available && \
    rm -rf /var/cache/apk/*

RUN adduser -g stockholmfootvolley -u 1890 -D stockholmfootvolley
COPY --from=builder  /usr/local/go/src/bin /home/stockholmfootvolley/bin
RUN chown -R stockholmfootvolley:stockholmfootvolley /home/stockholmfootvolley
USER 1890
WORKDIR /home/stockholmfootvolley

ENV PATH="/home/stockholmfootvolley/bin:${PATH}"
