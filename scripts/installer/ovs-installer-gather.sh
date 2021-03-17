#!/usr/bin/env bash

if test "x${1}" = 'x--id'
then
	GATHER_ID="${2}"
	shift 2
fi

ARTIFACTS="/tmp/artifacts-${GATHER_ID}"
mkdir -p "${ARTIFACTS}"

exec &> >(tee "${ARTIFACTS}/ovs-gather.log")

echo "Gather remote logs"
export MASTERS=()
if [[ -f ${LOG_BUNDLE_BOOTSTRAP_ARCHIVE_NAME} ]]; then
    # run on the current node
    MASTER_GATHER_ID="master-${GATHER_ID}"
    MASTER_ARTIFACTS="/tmp/artifacts-${MASTER_GATHER_ID}"
    mkdir -p "${MASTER_ARTIFACTS}/journals"
    mkdir -p "${ARTIFACTS}/control-plane/master"
    sudo /usr/local/bin/ovs-master-installer-gather.sh --id "${MASTER_GATHER_ID}" </dev/null
elif [ "$#" -ne 0 ]; then
    MASTERS=( "$@" )
elif test -s "${ARTIFACTS}/resources/masters.list"; then
    mapfile -t MASTERS < "${ARTIFACTS}/resources/masters.list"
else
    echo "no masters"
fi

for master in "${MASTERS[@]}"
do
    echo "Collecting info from ${master}"
    scp -o PreferredAuthentications=publickey -o StrictHostKeyChecking=false -o UserKnownHostsFile=/dev/null -q -r /usr/local/bin/ovs-master-installer-gather.sh "core@[${master}]:"
    ssh -o PreferredAuthentications=publickey -o StrictHostKeyChecking=false -o UserKnownHostsFile=/dev/null "core@${master}" -C "./ovs-master-installer-gather.sh --id '${GATHER_ID}'" </dev/null
done