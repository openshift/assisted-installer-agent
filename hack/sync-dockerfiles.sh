#!/bin/bash

set -euo pipefail

DOCKERFILE_1=Dockerfile.ocp
DOCKERFILE_2=Dockerfile.assisted_installer_agent

# Files that don't need to be in both (one per line)
EXCLUDE_MATCH_REGEX="agent-tui"
BINARIES_LIST_1=$(awk -v excludepattern="$EXCLUDE_MATCH_REGEX" '/COPY --from=builder/ && /\/build\// && $0 !~ excludep
attern {print $4}' "$DOCKERFILE_1" | sort)
BINARIES_LIST_2=$(awk -v excludepattern="$EXCLUDE_MATCH_REGEX" '/COPY --from=builder/ && /\/build\// && $0 !~ excludep
attern {print $4}' "$DOCKERFILE_2" | sort)

# Make sure that the same agent binaries are being copied in both Dockerfiles
echo Calculating diff...
if ! diff --side-by-side <(cat <<<"$BINARIES_LIST_1") <(cat <<<"$BINARIES_LIST_2"); then
    echo
    echo "ERROR: Both \"$DOCKERFILE_1\" and \"$DOCKERFILE_2\" must copy the same binaries, but the above diff has been found"
    exit 1
else
    echo "OK: Lists match"
    exit 0
fi
