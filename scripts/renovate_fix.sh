#!/usr/bin/bash

if [[ "$GITHUB_HEAD_REF" == "renovate/"* ]]; then
  go mod tidy
fi
