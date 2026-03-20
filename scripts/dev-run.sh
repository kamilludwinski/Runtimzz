#!/usr/bin/env bash

set -e

cd "$(dirname "$0")/.."
export DEV_APP_NAME=Runtimz-dev

exec go run . "$@"
