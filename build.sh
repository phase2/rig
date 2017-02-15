#!/bin/bash
#
# To use this you will need to: go get github.com/mitchellh/gox

go get github.com/tools/godep
go get github.com/mitchellh/gox

godep restore

# Build for mac
gox -cgo -osarch="Darwin/amd64" -output="build/darwin/rig"

# Build for windows and linux
gox -osarch="Linux/amd64" -osarch="Windows/amd64" -output="build/{{.OS}}/rig"

