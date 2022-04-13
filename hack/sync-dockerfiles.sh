#!/bin/bash

set -euo pipefail

DOCKERFILE_1=Dockerfile.ocp
DOCKERFILE_2=Dockerfile.assisted_installer_agent

TMP_BINARIES_LIST_1=$(mktemp)
TMP_BINARIES_LIST_2=$(mktemp)

echo "Searching for binary copies in \"$DOCKERFILE_1\""
cat $DOCKERFILE_1 | \
        grep 'COPY --from=builder' | grep '/build/' | rev | cut -d' ' -f1 | rev | sort | tee $TMP_BINARIES_LIST_1
echo

echo "Searching for binary copies in \"$DOCKERFILE_2\""
cat $DOCKERFILE_2 | \
        grep 'COPY --from=builder' | grep '/build/' | rev | cut -d' ' -f1 | rev | sort | tee $TMP_BINARIES_LIST_2
echo

# Make sure that the same agent binaries are being copied in both Dockerfiles
echo Calculating diff...
if ! diff "$TMP_BINARIES_LIST_1" "$TMP_BINARIES_LIST_2"; then
    echo
    echo "ERROR: Both \"$DOCKERFILE_1\" and \"$DOCKERFILE_2\" must copy the same binaries, but the above diff has been found"
    exit 1
else
    echo "OK: Lists match"
    exit 0
fi
