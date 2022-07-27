#!/bin/bash
set -e
tesbted_name="$1"

if [ -z "${tesbted_name}" ]; then
  echo "Usage: $0 <testbed_name>"
  exit 1
fi

if [ ! -e "terraform.tfstate.d/${tesbted_name}" ]; then
  echo "${tesbted_name} doesn't exist in local workspace"
  exit 1
fi

echo ====== Terraform Output ======
terraform show "terraform.tfstate.d/${tesbted_name}/terraform.tfstate" | awk '/^Outputs:$/{show=1}{if(show==1)print $0}'
