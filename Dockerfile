FROM golang:1.9-alpine

RUN apk add --no-cache \
  ca-certificates \
  git \
  gcc \
  musl-dev \
  && go get -u github.com/golang/dep/... \
  && go get -u github.com/mitchellh/gox
