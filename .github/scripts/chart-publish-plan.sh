#!/usr/bin/env bash
# Decides whether the chart-publish workflow pushes the chart, and which
# appVersion it stamps into it.
#
# The appVersion always comes from a release tag, never from Chart.yaml, so a
# published chart never points at an image that has not been released yet:
#   - workflow_run (after a tag release): the tag that CI released.
#   - push / workflow_dispatch: the latest v* tag reachable from HEAD.
#
# A chart version is published once. On push, publishing waits for the next
# release when main holds app changes the latest tag does not have.
#
# Inputs (env): EVENT_NAME, CHART_DIR, CHART_REF, and RELEASE_TAG for workflow_run.
# Outputs ($GITHUB_OUTPUT): publish, chart_version, app_version.
set -euo pipefail

: "${EVENT_NAME:?EVENT_NAME is required}"
: "${CHART_DIR:?CHART_DIR is required}"
: "${CHART_REF:?CHART_REF is required}"
output="${GITHUB_OUTPUT:-/dev/stdout}"

# Paths that end up in the released binary or image.
app_paths=('*.go' go.mod go.sum Dockerfile .goreleaser.yaml)

chart_version=$(yq '.version' "${CHART_DIR}/Chart.yaml")

if [[ "${EVENT_NAME}" == workflow_run ]]; then
  tag="${RELEASE_TAG:?RELEASE_TAG is required for workflow_run}"
  if ! git tag --points-at HEAD | grep -qxF -- "${tag}"; then
    echo "::error::Tag ${tag} does not point at the checked-out commit $(git rev-parse --short HEAD)."
    exit 1
  fi
else
  tag=$(git describe --tags --abbrev=0 --match 'v*')
fi
app_version="${tag#v}"

publish=true
if published=$(helm show chart "${CHART_REF}" --version "${chart_version}" 2>&1); then
  publish=false
  published_app=$(yq '.appVersion' <<<"${published}")
  if [[ "${published_app}" == "${app_version}" ]]; then
    echo "::notice::Chart ${chart_version} is already published with appVersion ${app_version}. Nothing to do."
  elif [[ "${EVENT_NAME}" == workflow_run ]]; then
    echo "::error::Chart ${chart_version} is already published with appVersion ${published_app}. Bump the version in ${CHART_DIR}/Chart.yaml to publish appVersion ${app_version}."
    exit 1
  else
    echo "::notice::Chart ${chart_version} is already published with appVersion ${published_app}. Skipping."
  fi
elif ! grep -q 'not found' <<<"${published}"; then
  echo "${published}" >&2
  echo "::error::Could not check whether chart ${chart_version} is published."
  exit 1
elif [[ "${EVENT_NAME}" != workflow_run ]] && ! git diff --quiet "${tag}" HEAD -- "${app_paths[@]}"; then
  publish=false
  echo "::notice::main has app changes that ${tag} does not include. Chart ${chart_version} is published after the next release."
fi

echo "Chart ${chart_version}, appVersion ${app_version} (from ${tag}), publish=${publish}"
{
  echo "publish=${publish}"
  echo "chart_version=${chart_version}"
  echo "app_version=${app_version}"
} >>"${output}"
