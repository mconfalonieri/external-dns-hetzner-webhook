#!/bin/bash

if [[ "$GITHUB_HEAD_REF" == "renovate/"* ]]; then
  go mod tidy
else
  echo "Not on a renovate branch." >&2
  exit 1
fi
