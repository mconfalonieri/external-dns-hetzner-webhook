#!/usr/bin/bash

export HURL_k8s_api_url=http://localhost:8070
export HURL_dns_mockserver_url=http://dns-mockserver.127.0.0.1.nip.io
export HURL_zone_id="11af3414-ebba-11e9-8df5-66fbe8zoneid"
export HURL_record_id="11af3414-ebba-11e9-8df5-66fbrecordid"

kubectl proxy --port=8070 &
pid=$!

cleanup() {
  exitCode=$?
  echo "exit code: $exitCode, cleaning up...now"
  hurl test/hurl/service_nodeport_cleanup.hurl --test
  kill $pid
  exit $exitCode
}

trap cleanup EXIT

set -e

sleep 2

mkdir -p build/reports/hurl
hurl test/hurl/service_nodeport.hurl --test --report-html build/reports/hurl/ --report-junit build/reports/hurl/junit.xml


