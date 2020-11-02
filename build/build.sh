#!/usr/bin/env bash

# Copyright 2020 The OpenEBS Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script builds the application from source for multiple platforms.
set -e

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/../" && pwd )"

# Change into that directory
cd "$DIR"

# Get the git commit
if [ -f $GOPATH/src/github.com/openebs/cstor-operators/GITCOMMIT ];
then
    GIT_COMMIT="$(cat $GOPATH/src/github.com/openebs/cstor-operators/GITCOMMIT)"
else
    GIT_COMMIT="$(git rev-parse HEAD)"
fi

# Set BUILDMETA based on travis tag
if [[ -n "$TRAVIS_TAG" ]] && [[ $TRAVIS_TAG != *"RC"* ]]; then
    echo "released" > BUILDMETA
fi

# Get the version details
#VERSION="$(cat $GOPATH/src/github.com/openebs/cstor-operators/VERSION)"

# Determine the current branch
CURRENT_BRANCH=""
if [ -z "${TRAVIS_BRANCH}" ];
then
  CURRENT_BRANCH=$(git branch | grep "\*" | cut -d ' ' -f2)
else
  CURRENT_BRANCH="${TRAVIS_BRANCH}"
fi

## Populate the version based on release tag
## If travis tag is set then assign it as VERSION and
## if travis tag is empty then mark version as ci
if [ -n "$TRAVIS_TAG" ]; then
    # Trim the `v` from the TRAVIS_TAG if it exists
    # Example: v1.10.0 maps to 1.10.0
    # Example: 1.10.0 maps to 1.10.0
    # Example: v1.10.0-custom maps to 1.10.0-custom
    VERSION="${TRAVIS_TAG#v}"
else
    ## Marking VERSION as current_branch-dev
    ## Example: master branch maps to master-dev
    ## Example: v1.11.x-ee branch to 1.11.x-ee-dev
    ## Example: v1.10.x branch to 1.10.x-dev
    VERSION="${CURRENT_BRANCH#v}-dev"
fi

echo "Building for ${VERSION} VERSION"

#VERSION=$(git describe --tags --always --dirty)

# Determine the arch/os combos we're building for
UNAME=$(uname)
ARCH=$(uname -m)
if [ "$UNAME" != "Linux" -a "$UNAME" != "Darwin" ] ; then
    echo "Sorry, this OS is not supported yet."
    exit 1
fi

if [ "$UNAME" = "Darwin" ] ; then
  XC_OS="darwin"
elif [ "$UNAME" = "Linux" ] ; then
  XC_OS="linux"
fi

if [ -z "${PNAME}" ];
then
    echo "Project name not defined"
    exit 1
fi

if [ -z "${CTLNAME}" ];
then
    echo "CTLNAME not defined"
    exit 1
fi

# Delete the old dir
echo "==> Removing old directory..."
rm -rf bin/${PNAME}/*
mkdir -p bin/${PNAME}/

# If its dev mode, only build for ourself
if [[ "${DEV}" ]]; then
    XC_OS=$(go env GOOS)
    XC_ARCH=$(go env GOARCH)
fi

# Build!
echo "==> Building ${CTLNAME} using $(go version)... "

GOOS="${XC_OS}"
GOARCH="${XC_ARCH}"
output_name="bin/${PNAME}/"$GOOS"_"$GOARCH"/"$CTLNAME

if [ $GOOS = "windows" ]; then
    output_name+='.exe'
fi

env GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0  go build ${BUILD_TAG} -ldflags \
    "-X github.com/openebs/cstor-operators/pkg/version.GitCommit=${GIT_COMMIT} \
    -X main.CtlName='${CTLNAME}' \
    -X github.com/openebs/cstor-operators/pkg/version.Version=${VERSION}"\
    -o $output_name \
    ./cmd/${CTLNAME}

echo ""

# Move all the compiled things to the $GOPATH/bin
GOPATH=${GOPATH:-$(go env GOPATH)}
case $(uname) in
    CYGWIN*)
        GOPATH="$(cygpath $GOPATH)"
        ;;
esac
OLDIFS=$IFS
IFS=: MAIN_GOPATH=($GOPATH)
IFS=$OLDIFS

# Create the gopath bin if not already available
mkdir -p ${MAIN_GOPATH}/bin/

# Copy our OS/Arch to the bin/ directory
DEV_PLATFORM="./bin/${PNAME}/$(go env GOOS)_$(go env GOARCH)"
for F in $(find ${DEV_PLATFORM} -mindepth 1 -maxdepth 2 -type f); do
    cp ${F} bin/${PNAME}/
    cp ${F} ${MAIN_GOPATH}/bin/
done

if [[ "x${DEV}" == "x" ]]; then
    # Zip and copy to the dist dir
    echo "==> Packaging..."
    for PLATFORM in $(find ./bin/${PNAME} -mindepth 1 -maxdepth 1 -type d); do
        OSARCH=$(basename ${PLATFORM})
        echo "--> ${OSARCH}"

        pushd "$PLATFORM" >/dev/null 2>&1
        zip ../${PNAME}-${OSARCH}.zip ./*
        popd >/dev/null 2>&1
    done
fi

# Done!
echo
echo "==> Results:"
ls -hl bin/${PNAME}/
