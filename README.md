# Rig - Outrigger CLI [![Build Status](https://travis-ci.org/phase2/rig.svg?branch=develop)](https://travis-ci.org/phase2/rig)

> A CLI for managing the Outrigger container-driven development stack.

See the [documentation for more details](http://docs.outrigger.sh).

Use this readme when you want to develop the Outrigger CLI.

Setup
-----

Install go from homebrew using the flag to include common cross-compiler targets (namely Darwin, Linux, and Windows) 

```bash
brew install go --with-cc-common
```

Setup `$GOPATH` and `$PATH` in your favorite shell (`~/.bashrc` or `~/.zshrc`) This assumes you have your code within your `$GOPATH`.

```bash
export GOPATH=$HOME/Projects
export PATH=$PATH:$GOPATH/bin
```

Checkout the code into your $GOPATH, likely in `$GOPATH/src/github.com/phase2/rig`

Get all the dependencies

```bash
# Go Dependency Manager
go get github.com/tools/godep

# Go Cross Platform Build Tool
go get github.com/mitchellh/gox

# Install the project dependencies into $GOPATH
cd $GOPATH/src/github.com/phase2/rig/cli
godep restore
```

Code
----

We make use of a few key libraries to do all the fancy stuff that the `rig` CLI will do.
 
 * https://github.com/urfave/cli
     * The entire CLI framework from helps text to flags. This was an easy cli to build b/c of this library 
 * https://github.com/fatih/color
     * All the fancy terminal color output
 * https://github.com/bitly/go-simplejson
     * The JSON parse and access library used primarily with the output of `docker-machine inspect` 
 * https://gopkg.in/yaml.v2
     * The YAML library for parsing/reading YAML files 

Build
-----

If you want to build `rig` for all platforms, simply execute the `build.sh` script from the root 
directory. This script will build the binary for Darwin, Linux and Windows and put it in the appropriate
 `dist/[PLATFORM]` directory for each operating system.

For development, sometimes you will just want to build for your target platform because it is faster. TO
do that, simply run the following command.

```bash
gox -osarch="Darwin/amd64" -output="build/{{.OS}}/rig"
```
   
This command targets an OS/Architecture (Darwin/Mac and 64bit) and puts the resultant file in the `bin/`
directory for the appropriate OS with the name `rig`.  

Developing Rig with Docker [Experimental]
-----------------------------------------

You can use the Docker integration within this repository to facilitate development in lieu of setting up a
local golang environment. Using docker-compose, run the following commands:

```bash
docker-compose run --rm install
docker-compose run --rm build
```

This will produce a working OSX binary at `build/darwin/rig`.

Deploy to Homebrew
------------------

We now manage the Mac / OSX version of the binaries via `brew`.  To publish a new build to `brew` you must
perform the following operations.

 - Change the code :)
 - When changing the code make sure to update the VERSION variable in main.go
 - Build all the code via `build.sh`
 - Make sure you have https://github.com/phase2/homebrew-outrigger cloned into ~/Projects/homebrew-outrigger
 - Prepare a new `brew` version via `brew-publish.sh`
    - Part of this will write a new formula into homebrew-outrigger
 - Commit & push the updated formula to publish the new version
