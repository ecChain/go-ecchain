#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
ecdir="$workspace/src/github.com/ecereum"
if [ ! -L "$ecdir/go-ecereum" ]; then
    mkdir -p "$ecdir"
    cd "$ecdir"
    ln -s ../../../../../. go-ecereum
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$ecdir/go-ecereum"
PWD="$ecdir/go-ecereum"

# Launch the arguments with the configured environment.
exec "$@"
