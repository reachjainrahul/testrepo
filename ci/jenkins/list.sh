#!/bin/bash
set -e

echo ====== Local Workspaces ======
terraform workspace list | grep -xv '^[* ]*default$'

echo ====== Shared Workspaces ======
ls -1 ../terraform.tfstate.d/current/ | awk '{print "  "$0}'
