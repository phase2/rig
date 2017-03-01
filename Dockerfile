FROM golang:1.7-alpine

RUN apk add --no-cache \
  ca-certificates \
  git \
  gcc \
  musl-dev \
  && go get github.com/tools/godep \
  && go get github.com/mitchellh/gox
