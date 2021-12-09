# GitHub Metric Extractor

Metric extractor for github repositories

## SDK Setup

### Go

* Leveraging Go Version 1.17 (<https://golang.org/dl/>)

### Linter

* Utilizing golangci-lint Version 1.42 (<https://github.com/golangci/golangci-lint>)

## Project Structure

Following standards published here...

* <https://github.com/golang-standards/project-layout>

## Build Process

Currently leveraging a custom build script.  Want to look at leveraging Gradle in the future but initial attempts ran into roadblocks with available plugins  either requiring Java 15, requiring Developer level access on windows with additional admin rights, or lacked sufficient documentation.  Didn't use Makefile because Make not available with standard git bash install.

Current build contains a bash script (`make.sh`) to enable the following functions...

* Building all packages (`make.sh` or `make.sh build`)
* Test specific package (`make.sh test <package : ./...>`)
* Cleaning build artifacts (`make.sh clean`)
* Linting files (`make.sh lint <pattern : ./...>`)

Test coverage will be calculated during testing and results can be viewed in the `cover.html` file in the bin directory.

## Usage

### Logging

Program leverages the [glog](https://github.com/golang/glog) library to enable leveled log support.  Command line arguments are available as well to output logs to stderr or files...

* `-logtostderr`: logs are written to standard error instead of files
* `-alsologtostderr`: logs to standard error in addition to files
* `-stderrthreshold ERROR`: log events at or above the level to standard error as well as files
* `-v 2`: log events at supplied level (DEBUG) or lower

Level practice used within the source...

* `2`: DEBUG level
* `3`: TRACE level

### GitHub Authorization

Program will expect `GITHUB_AUTH_TOKEN` to be populated with an access token that has READ access for...

* User
* Repositories

### GitHub Base URL

Program will assume a base GitHub Enterprise url of `https:\\github.edwardjones.com\` but can be overriden using the `baseURL` flag.

### Base Data Directory

Program will assume a base data directory of `.git-metrics` under the users home directory (current working directory if the user home directory can't be located) but can be overriden using the `dataDir` flag.

### Command Examples

Update Metrics For All Repositories (using default org of `ej`) changed since last update: Logs to Stderr and Debug Level On

```bash
./git-what update-metrics --logtostderr --v 2
```

Update Metrics For All Repositories (using default org of `ej`) regardless of last update: Logs to Stderr

```bash
./git-what update-metrics --forceUpdate --logtostderr
```

Update Metrics For All Repositories using specific organization

```bash
./git-what update-metrics --org ejcodefest --logtostderr
```

Update Metrics for a particular repository

```bash
./git-what update-metrics --repo enterprise-arch --logtostderr
```
