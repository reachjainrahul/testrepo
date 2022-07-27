#!/bin/bash
set -e

tesbted_name="$1"

if [ -z "${tesbted_name}" ]; then
  echo "Usage: $0 <testbed_name>"
  exit 1
fi

if [ ! -e "terraform.tfstate.d/${tesbted_name}" ]; then
  echo "${tesbted_name} does not exist in local workspace"
  exit 1
fi

if [ ! -e "../terraform.tfstate.d/current/${tesbted_name}" ]; then
  echo "${tesbted_name} does not exist in remote workspace"
  exit 1
fi

echo ====== Showing Differences for ${tesbted_name} between Local and Shared Workspaces======
diff -sur "terraform.tfstate.d/${tesbted_name}" "../terraform.tfstate.d/current/${tesbted_name}"
