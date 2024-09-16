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

check_git_status
go mod tidy
make fmt
check_git_status

make static-analysis

# License check fails with:
#
#    save.go:125:9: cannot use func(src string) (bool, error) {…} (value of type
#    func(src string) (bool, error)) as func(srcinfo fs.FileInfo, src string,
#    dest string) (bool, error) value in struct literal
#
# Will report bug to Google. In the meantime the check is disabled.
# make license-check
