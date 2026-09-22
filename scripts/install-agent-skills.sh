#!/usr/bin/env bash

# Installs the YewSeal agent skills into ./.agents/skills/ of the current
# directory, fetching only the files under skills/ from the repository tag
# that matches the installed yews binary. Requires an authenticated gh CLI.

set -euo pipefail

repo="YewFence/YewSeal"
dest="${PWD}/.agents/skills"

version="$(yews --version | awk '{print $3}')"
if [[ ! "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Could not parse a release version from 'yews --version' (got '${version}')."
  echo "A release build is required; a locally built binary has no matching tag."
  exit 1
fi
ref="v${version}"

staging="$(mktemp -d)"
while IFS= read -r path; do
  target="${staging}/${path#skills/}"
  mkdir -p "$(dirname "${target}")"
  gh api "repos/${repo}/contents/${path}?ref=${ref}" \
    -H "Accept: application/vnd.github.raw" > "${target}"
done < <(gh api "repos/${repo}/git/trees/${ref}?recursive=1" \
  --jq '.tree[] | select(.type == "blob" and (.path | startswith("skills/"))) | .path')

mkdir -p "${dest}"
rm -rf "${dest}/yewseal" "${dest}/yewseal-secrets"
mv "${staging}/yewseal" "${staging}/yewseal-secrets" "${dest}/"

echo "Installed YewSeal agent skills ${ref} into ${dest}"
echo "Re-run after upgrading the CLI to refresh them."
