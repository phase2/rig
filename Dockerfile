FROM golang:1.9

# Install Ruby
WORKDIR /tmp
RUN apt-get -y update \
    && apt-get -y install build-essential zlib1g-dev libssl-dev libreadline6-dev libyaml-dev
RUN wget https://cache.ruby-lang.org/pub/ruby/2.4/ruby-2.4.2.tar.gz \
    && tar xzf ruby-2.4.2.tar.gz \
    && cd ruby-2.4.2 \
    && ./configure --prefix=/usr/local \
    && make \
    && make install

# Install fpm for package building
RUN apt-get install -y rpm \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists \
    && gem install --no-rdoc --no-ri fpm

# Back to the Go thingies
WORKDIR /go
RUN go get -u github.com/golang/dep/... \
  && go get -u github.com/alecthomas/gometalinter \
  && go get -u github.com/goreleaser/goreleaser
RUN gometalinter --install --update
