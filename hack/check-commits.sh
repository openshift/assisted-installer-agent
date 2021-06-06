#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

parent_branch="master"
git remote add openshift https://github.com/openshift/assisted-installer-agent.git &> /dev/null || true
git fetch openshift ${parent_branch}

current_branch="$(git rev-parse --abbrev-ref HEAD)"

revs=$(git rev-list "FETCH_HEAD".."${current_branch}")

for commit in ${revs};
do
    commit_message=$(git cat-file commit ${commit} | sed '1,/^$/d')
    tmp_commit_file="$(mktemp)"
    echo "${commit_message}" > ${tmp_commit_file}
    ${__dir}/check-commit-message.sh "${tmp_commit_file}"
done


exit 0
