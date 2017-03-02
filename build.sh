#!/bin/bash
#
# To use this you will need to: go get github.com/mitchellh/gox

go get github.com/tools/godep
go get github.com/mitchellh/gox

pushd cli

godep restore

# Build for platforms
gox -osarch="Darwin/amd64" -output="../build/darwin/rig"
gox -osarch="Linux/amd64" -output="../build/linux/rig"
gox -osarch="Windows/amd64" -output="../build/windows/rig"

popd

# Get version
VERSION=`build/darwin/rig --version 2>&1 | grep 'rig version' | awk '{print $3}'`

# Create archives for publishing to GitHub
cp scripts/docker-machine-watch-rsync.sh build/darwin/.
pushd build/darwin
tar czf ../rig-${VERSION}-darwin-amd64.tar.gz rig docker-machine-watch-rsync.sh
popd
pushd build/linux
tar czf ../rig-${VERSION}-linux-amd64.tar.gz rig
popd
pushd build/windows
zip ../rig-${VERSION}-windows-amd64.zip rig.exe
popd