# CONTRIBUTING

Thank you for considering contributing to the Outrigger CLI!

## Quality Contributions

* Make sure your branch will compile.
* Make sure your branch passes our static analysis checks.
* Make sure your branch conforms with go fmt standards.
* Manually test your changes.

## User Interactions

One of the key goals of this project is to promote a positive developer
experience. Every interaction should be thought of with the following points:

* Are you providing the user with enough context about what's being asked or being done?
* Does the user expect to wait? Might the user think the tool stalled?
* Is there black box business happening that could be made more transparent?

We have a slightly complex logging API to support addressing these concerns.
(See ./util/logging.go)

Here are a few conventions:

* **Starting a task that could take more than 5 seconds:**
  * `cmd.out.Spin("Preparing the sauce")`
* **Use the correct method to log operational results: (Pick one)**
  * `cmd.out.Info("Sauce is Ready.")`
  * `cmd.out.Warning("Sauce is burnt on the bottom.")`
  * `cmd.out.Error("Discard this sauce and try again.")`
* **Going to send some contextual notes to the user**:
  1. `cmd.out.NoSpin()` if currently using the spinner.
  2. `cmd.out.Info("Sauce exists.")`
  4. `cmd.out.Verbose("The ingredients of the sauce include tomato, salt, black pepper, garlic...")`
* **Command has executed and is successful. Please no notification:**
  ```
  cmd.out.Info("Enjoy your dinner.")
  return cmd.Success("")
  ```
* **Command has executed and is successful. Get a notification too!**
  ```
  return cmd.Success("Enjoy your dinner.")
  ```
* **Command failed:**
  ```
  message := "Cooking sauce is hard, we failed"
  cmd.out.Error("%s: %s", message, err.Error())
  return cmd.Failure(message)
  ```

## Developer Testing Commands

You can use `rig dev:win` or `rig dev:fail` as no-op commands to observe the
effects of a success or failure without external dependencies on the local
environment or side effects from "real" commands doing their job.

## Development Environment Setup

### Developing with Docker

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

If you want to run the static analysis checks:

```bash
docker-compose run --rm lint
```

If you want to run go fmt against the codebase:
```bash
docker-compose run --rm base go fmt ./...
```

### Developing Locally

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
cd $GOPATH/src/github.com/phase2/rig
dep ensure
```

#### Building Rig

If you want to build  `rig` locally for your target platform, simply run the following command:

```bash
GOARCH=amd64 GOOS=darwin go build -o build/darwin/rig cmd/main.go
```

This command targets an OS/Architecture (Darwin/Mac and 64bit) and puts the resultant file in the `build/darwin/`
with the name `rig`.  Change `GOARCH` and `GOOS` if you need to target a different platform
