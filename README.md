# Rig - Outrigger CLI [![Build Status](https://travis-ci.org/phase2/rig.svg?branch=develop)](https://travis-ci.org/phase2/rig)

> A CLI for managing the Outrigger's container-driven development stack.

See the [documentation for more details](http://docs.outrigger.sh).
See the [CONTRIBUTING.md](./CONTRIBUTING.md) for developer documentation.

## Built on Dependencies

We make use of a few key libraries to do all the fancy stuff that the `rig` CLI will do.

 * https://github.com/urfave/cli
     * The entire CLI framework from helps text to flags.
     This was an easy cli to build b/c of this library.
 * https://github.com/fatih/color
     * All the fancy terminal color output
 * https://github.com/bitly/go-simplejson
     * The JSON parse and access library used primarily with the output
     of `docker-machine inspect`
 * https://gopkg.in/yaml.v2
     * The YAML library for parsing/reading YAML files
 * https://github.com/martinlindhe/notify
     * Cross-platform desktop notifications

## Release Intructions

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
