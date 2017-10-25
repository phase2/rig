# Rig - Outrigger CLI [![Build Status](https://travis-ci.org/phase2/rig.svg?branch=develop)](https://travis-ci.org/phase2/rig)

> A CLI for managing the Outrigger container-driven development stack.

See the [documentation for more details](http://docs.outrigger.sh).

Use this readme when you want to develop the Outrigger CLI.

Setup
------

Install go from homebrew using the flag to include common cross-compiler targets (namely Darwin, Linux, and Windows) 

```bash
brew install go --with-cc-common
brew install dep
brew tap goreleaser/tap
brew install goreleaser/tap/goreleaser
```

Setup `$GOPATH` and `$PATH` in your favorite shell (`~/.bashrc` or `~/.zshrc`)

```bash
export GOPATH=$HOME/Projects
export PATH=$PATH:$GOPATH/bin
```

Checkout the code into your `$GOPATH` in `$GOPATH/src/github.com/phase2/rig`

Get all the dependencies

```bash
# Install the project dependencies into $GOPATH
cd $GOPATH/src/github.com/phase2/rig/cli
dep ensure
```

Developing Locally
-------------------

If you want to build  `rig` locally for your target platform, simply run the following command:

```bash
GOARCH=amd64 GOOS=darwin go build -o ../build/darwin/rig
```
   
This command targets an OS/Architecture (Darwin/Mac and 64bit) and puts the resultant file in the `build/darwin/`
with the name `rig`.  Change `GOARCH` and `GOOS` if you need to target a different platform

Developing with Docker
-----------------------

You can use the Docker integration within this repository to facilitate development in lieu of setting up a
local golang environment. Using docker-compose, run the following commands:

```bash
docker-compose run --rm install
docker-compose run --rm compile
```

This will produce a working OSX binary at `build/darwin/rig`.

If you change a dependency in `Gopkg.toml` you can update an individual package dependency with:

```bash
docker-compose run --rm update [package]
```

If you want to update all packages use:

```bash
docker-compose run --rm update
```


Release
-------

We use [GoReleaser](https://goreleaser.com) to handle nearly all of our release concerns.  GoReleaser will handle

* Building for all target platforms
* Create a GitHub release on our project page based on tag
* Create archive file for each target platform and attach it to the GitHub release
* Update the Homebrew formula and publish it
* Create .deb and .rpm packages for linux installations

To create a new release of rig:
* Get all the code committed to `master`
* Tag master with the new version number
* Run `docker-compose run --rm goreleaser`
* ...
* Profit!


Dependencies
-------------

We make use of a few key libraries to do all the fancy stuff that the `rig` CLI will do.
 
 * https://github.com/urfave/cli
     * The entire CLI framework from helps text to flags. This was an easy cli to build b/c of this library 
 * https://github.com/fatih/color
     * All the fancy terminal color output
 * https://github.com/bitly/go-simplejson
     * The JSON parse and access library used primarily with the output of `docker-machine inspect` 
 * https://gopkg.in/yaml.v2
     * The YAML library for parsing/reading YAML files 
