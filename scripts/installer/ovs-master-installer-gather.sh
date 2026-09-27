#!/usr/bin/env bash

if test "x${1}" = 'x--id'
then
	GATHER_ID="${2}"
	shift 2
fi

ARTIFACTS="/tmp/artifacts-${GATHER_ID}"
mkdir -p "${ARTIFACTS}"

echo "Gathering master ovs journals ..."
mkdir -p "${ARTIFACTS}/journals"
for service in ovs-vswitchd ovsdb-server ovs-configuration
do
    journalctl --boot --no-pager --output=short --unit="${service}" > "${ARTIFACTS}/journals/${service}.log"
done

echo "Waiting for logs ..."
wait