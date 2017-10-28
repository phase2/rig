#!/usr/bin/env sh
##
# Test go fmt
#
# Test if the codebase has valid go formatting.
#
# @see https://github.com/limetext/lime/pull/265/files
##

fmt="$(find . -not -path '*/vendor/*' -type f -name '*.go' -print0 | xargs -0 gofmt -l )"

if [ -n "$fmt" ]; then
    echo "Unformatted Go source code:"
    echo "$fmt"
    exit 1
fi
