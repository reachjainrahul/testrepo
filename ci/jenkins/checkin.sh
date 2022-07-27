#!/bin/bash
set -e

tesbted_name="$1"
force="$2"

if [ -z "${tesbted_name}" ]; then
  echo "Usage: $0 <testbed_name> [-f]"
  exit 1
fi

if [ ! -e "terraform.tfstate.d/${tesbted_name}" ]; then
  echo "${tesbted_name} does not exist in local workspace"
  exit 1
fi

if [ -e "../terraform.tfstate.d/current/${tesbted_name}" ]; then
  if [ "${force}" != "-f" ]; then
    echo "Shared workspace already exists, use $0 <testbed_name> [-f] to force overwrite."
    exit 1
  fi
fi

if [ ! -e "../terraform.tfstate.d/current/" ]; then
  mkdir -p ../terraform.tfstate.d/current/
fi

echo ====== Checking in ${tesbted_name} to Shared Workspace ======
cp -rf "terraform.tfstate.d/${tesbted_name}" "../terraform.tfstate.d/current/"
echo ====== Done ======
