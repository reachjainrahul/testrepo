# Copyright 2022 Antrea Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# define the cleanup_testbed function
#!/bin/bash
set -ex

buildNumber=""
vcHost=""
vcUser=""
dataCenterName=""
dataStore=""
vcCluster=""
resourcePoolPath=""
vcNetwork=""
virtualMachine=""
goVcPassword=""
testUserName=""

_usage="Usage: $0 [--buildnumber <jenkins BUILD_NUMBER>] [--vchost <VC IPaddress/Domain Name>] [--vcuser <VC username>]
                  [--datacenter <datacenter to deploy vm>] [--datastore <dataStore name>] [--vcCluster <clusterName to deploy vm>]
                  [--resourcePool <resourcePool name>] [--vcNetwork <network used to delpoy vm>] [--virtualMachine <vm template>]
                  [--goVcPassword <Password for VC>] [--testUserName <a sample variable for test-aws.sh>]
Setup a VM to run antrea cloud e2e tests.
        --buildnumber           A number that is used to distinguish vm name from others.
        --vchost                VC ipAddress or domain name to deploy vm.
        --vcuser                User name for VC.
        --goVcPassword          Password to the user name for VC.
        --datacenter            Data center that is used to deploy vm.
        --datastore             Data store that is used to deploy vm.
        --vcCluster             VC cluster that is used to deploy vm.
        --resourcePool          Resource pool that is used to deploy vm.
        --vcNetwork             Network that is used to deploy vm.
        --virtualMachine        VM template that is used to deploy vm.
        --testUserName          A sample environment variable for test-aws.sh."

function echoerr {
    >&2 echo "$@"
}

function print_usage {
    echoerr "$_usage"
}

function print_help {
    echoerr "Try '$0 --help' for more information."
}

while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    --buildnumber)
    buildNumber="$2"
    shift 2
    ;;
    --vchost)
    vcHost="$2"
    shift 2
    ;;
    --vcuser)
    vcUser="$2"
    shift 2
    ;;
    --datacenter)
    dataCenterName="$2"
    shift 2
    ;;
    --datastore)
    dataStore="$2"
    shift 2
    ;;
    --vcCluster)
    vcCluster="$2"
    shift 2
    ;;
    --vcNetwork)
    vcNetwork="$2"
    shift 2
    ;;
    --resourcePool)
    resourcePoolPath="$2"
    shift 2
    ;;
    --virtualMachine)
    virtualMachine="$2"
    shift 2
    ;;
    --goVcPassword)
    goVcPassword="$2"
    shift 2
    ;;
    --testUserName)
    testUserName="$2"
    shift 2
    ;;
esac
done

cd ci/jenkins
if [ ! -e id_rsa ]; then
  ssh-keygen -t rsa -P '' -f id_rsa
fi
if [ ! -e playbook/jenkins_id_rsa ];then
  ssh-keygen -t rsa -P '' -f playbook/jenkins_id_rsa
fi
chmod 0600 id_rsa
chmod 0600 playbook/jenkins_id_rsa
testbed_name="antrea-test-cloud-${buildNumber}"

sudo apt install -y unzip ansible

cat terraform-${vcHost}.tfvars
rm -rf terraform-${vcHost}.tfvars
cat << EOF > terraform-${vcHost}.tfvars
vsphere_user="${vcUser}"
vm_count=1
vsphere_datacenter="${dataCenterName}"
vsphere_datastore="${dataStore}"
vsphere_compute_cluster="${vcCluster}"
vsphere_resource_pool="${resourcePoolPath}"
vsphere_network="${vcNetwork}"
vsphere_virtual_machine="${virtualMachine}"
EOF
cat terraform-${vcHost}.tfvars

./deploy.sh ${testbed_name} ${vcHost} ${goVcPassword}

ip_addr=`cat terraform.tfstate.d/${testbed_name}/terraform.tfstate | jq -r .outputs.vm_ips.value[0]`
chmod 0600 id_rsa
ssh -i id_rsa ubuntu@${ip_addr} "sudo apt-get update -y && sudo apt-get install -y ca-certificates curl unzip gnupg lsb-release"
scp -i id_rsa test-aws.sh ubuntu@${ip_addr}:~
function cleanup_testbed() {
  echo "=== retrieve logs ==="
  scp -i id_rsa ubuntu@${ip_addr}:~/test.log ../../..

   echo "=== cleanup vm ==="
   ./destroy.sh "${testbed_name}" "${goVcPassword}"

   cd ../../..
  tar zvcf test.log.tar.gz test.log
}

trap cleanup_testbed EXIT
ssh -i id_rsa ubuntu@${ip_addr} "chmod +x ./test-aws.sh; ./test-aws.sh ${testUserName}"
