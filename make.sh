#!/bin/bash
set -e
set -u
#--------------------------------------------------------------------------------------
# Rudimentary BUILD script since neither Make or Gradle usage on windows was working
#
# Usage:
#   ./make.sh            : Builds all packages
#   ./make.sh clean      : Cleans build artifacts
#   ./make.sh build      : Builds the project
#   ./make.sh lint <file>: Lints specified path (all when not supplied)
#   ./make.sh test <path>: Runs go test for for path (all when not supplied)
#--------------------------------------------------------------------------------------
GOLANGCI_LINT_VER="1.42.1"
BUILD_OUTPUT_DIR="bin"

# Process script arguments
RUNTYPE=${1:-"build"}
FILTER=${2:-"./..."}

# Clean all build artifacts
clean() {
    echo " "
    echo "*** CLEAN ALL BUILD ARTIFACTS..."
    echo " "
    rm -rf "./$BUILD_OUTPUT_DIR"
}

# full build
build() {
    lint "$1"
    test "$1"
    compile "$1"
}

# compile specified packages
compile() {
    echo " "
    echo "*** COMPILING PACKAGE(S): $1"
    echo " "
    GIT_VERSION_PATH=github.edwardjones.com/ej/ea-github-metric-extractor/pkg/version.gitVersion
    GIT_VERSION=$(git describe --abbrev=0 --tags | cut -b 2-)

    if [ "$GIT_VERSION"  == "" ]; then
      echo "no git version available, default to 0.1"
      GIT_VERSION="0.1"
    fi
    GIT_COMMIT_PATH=github.edwardjones.com/ej/ea-github-metric-extractor/pkg/version.gitCommit
    GIT_COMMIT=$(git rev-parse HEAD | cut -b -8)
    SOURCE_DATE_EPOCH=$(git show -s --format=format:%ct HEAD)
    BUILD_DATE_PATH=github.edwardjones.com/ej/ea-github-metric-extractor/pkg/version.buildDate
    DATE_FMT="%Y-%m-%dT%H:%M:%SZ"
    BUILD_DATE=$(date -u -d "@$SOURCE_DATE_EPOCH" "+${DATE_FMT}" 2>/dev/null || date -u -r "${SOURCE_DATE_EPOCH}" "+${DATE_FMT}" 2>/dev/null || date -u "+${DATE_FMT}")
    LDFLAGS="-X ${GIT_VERSION_PATH}=${GIT_VERSION} -X ${GIT_COMMIT_PATH}=${GIT_COMMIT} -X ${BUILD_DATE_PATH}=${BUILD_DATE}"

    echo "$LDFLAGS"
    go build -ldflags "${LDFLAGS}" -o "./$BUILD_OUTPUT_DIR/" "$1"
    retVal=$?
    if [ $retVal -ne 0 ]; then
        echo "Build errors detected in path: $1"
    fi
    return $retVal
}

# test specified packages
test() {
    echo " "
    echo "*** Testing PACKAGE(S): $1"
    echo " "
    go test -coverprofile "$BUILD_OUTPUT_DIR/cover.out" "$1"
    go tool cover -html="$BUILD_OUTPUT_DIR/cover.out" -o "$BUILD_OUTPUT_DIR/cover.html"
    retVal=$?
    if [ $retVal -ne 0 ]; then
        echo "Test errors detected in path: $1"
    fi
    return $retVal
}

# Linting
lint() {
    echo " "
    echo "*** Linting With Pattern: $1"
    echo " "
    linter_check
    ./$BUILD_OUTPUT_DIR/golangci-lint run "$1"
    retVal=$?
    if [ $retVal -ne 0 ]; then
        echo "Linting errors detected"
    fi
    return $retVal
}

linter_check() {
    version=""
    if command -v ./$BUILD_OUTPUT_DIR/golangci-lint &> /dev/null; then
        version="$(./$BUILD_OUTPUT_DIR/golangci-lint version --format short 2>&1 || true)"
    fi

    if [ "$version" != "$GOLANGCI_LINT_VER" ]; then
        echo "golangci-lint missing or not version '${GOLANGCI_LINT_VER}', downloading..."
        curl -sSfLk "https://raw.githubusercontent.com/golangci/golangci-lint/v${GOLANGCI_LINT_VER}/install.sh" | sh -s -- -b ./$BUILD_OUTPUT_DIR "v${GOLANGCI_LINT_VER}"
    fi
}

# Execute functions based on run type
retVal=0
case "$RUNTYPE" in
    clean)
        clean
        ;;
    compile)
        compile "$FILTER"
        retVal=$?
        ;;
    build)
        build "$FILTER"
        retVal=$?
        ;;
    lint)
        lint "$FILTER"
        retVal=$?
        ;;
    test)
        test "$FILTER"
        retVal=$?
        ;;
    *)
        echo "Unknown run type supplied"
        retVal=1
esac

exit $retVal
