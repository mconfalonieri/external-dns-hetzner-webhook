#!/usr/bin/bash

set -e # exit on first error

function check_git_status() {
if [[ -z $(git status --porcelain) ]]
then
  echo "ok"
else
  echo "Found modified files:"
  git status --porcelain
  exit 1
fi
}

if [[ -z "$GITHUB_REF_NAME" ]]
then
  echo "Running locally"
else
  echo "Branch: $GITHUB_REF_NAME"
fi
check_git_status
go mod tidy
make fmt
check_git_status

make static-analysis

# make license-check
