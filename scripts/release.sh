#!/bin/bash

# Conditional assignment of NIGHTLY_FLAG
if [ "$NIGHTLY" = "true" ]; then
  NIGHTLY_FLAG="--nightly"
else
  NIGHTLY_FLAG=""
fi

# Docker run command with Bash variables
docker run \
  --rm \
  -e CGO_ENABLED=1 \
  -e GORELEASER_KEY="$GORELEASER_KEY" \
  -e GITHUB_TOKEN="$GITHUB_TOKEN" \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "$HOME/.docker/config.json":/root/.docker/config.json \
  -v "$(pwd)":/go/src/"$PACKAGE_NAME" \
  -v "$(pwd)"/sysroot:/sysroot \
  -w /go/src/"$PACKAGE_NAME" \
  ghcr.io/goreleaser/goreleaser-cross-pro:v1.19.5 \
  release --clean --auto-snapshot $NIGHTLY_FLAG
