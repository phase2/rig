FROM golang:1.9-alpine

RUN apk add --no-cache \
  ca-certificates \
  git \
  gcc \
  libffi-dev \
  make \
  musl-dev \
  rpm \
  ruby \
  ruby-dev \
  tar \
  && go get -u github.com/golang/dep/... \
  && go get -u github.com/goreleaser/goreleaser

RUN gem install --no-rdoc --no-ri fpm