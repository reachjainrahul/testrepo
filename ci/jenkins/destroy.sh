#!/bin/bash
set -e

tesbted_name="$1"
var_file="terraform.tfstate.d/${tesbted_name}/vars.tfvars"

if [ -z "${tesbted_name}" ]; then
  echo "Usage: $0 <testbed_name>"
  exit 1
fi

if [ ! -e ".terraform" ]; then
  terraform init
fi

if [ ! -e "terraform.tfstate.d/${tesbted_name}" ]; then
  echo "${tesbted_name} doesn't exist in local workspace"
  exit 1
fi

echo ====== Deleting ${tesbted_name} from Local Workspace ======
terraform workspace "select" "${tesbted_name}"
source ${var_file}
terraform destroy -lock=false -auto-approve -var-file=terraform-${vsphere_server}.tfvars "-var-file=${var_file}" -parallelism=20
terraform workspace "select" default
terraform workspace delete "${tesbted_name}"
echo ====== Deleted ${tesbted_name} from Local Workspace ======
echo ====== Deleting ${tesbted_name} from Shared Workspace ======
rm -rf "../terraform.tfstate.d/current/${tesbted_name}"
echo ====== Deleted ${tesbted_name} from Shared Workspace ======
