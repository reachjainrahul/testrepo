#!/usr/bin/env bash

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

KUBECTL_VERSION=v1.24.1
TERRAFORM_VERSION=0.13.5

echo "Installing kubectl"
curl -LO https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
chmod +x ./kubectl && sudo mv ./kubectl /usr/local/bin/kubectl

echo "Installing Terraform"
sudo apt-get install unzip
curl -Lo ./terraform.zip https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip
unzip ./terraform.zip
chmod +x ./terraform && sudo mv ./terraform /usr/local/bin/terraform

echo "Installing AWS CLI"
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install

sudo apt-get install -y pv bzip2 jq

curl --silent --location "https://github.com/weaveworks/eksctl/releases/latest/download/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp
chmod +x /tmp/eksctl && sudo mv /tmp/eksctl /usr/local/bin

KEY_PAIR="nephe-$$"

export TF_VAR_aws_access_key_id=$1
export TF_VAR_aws_access_key_secret=$2
export TF_VAR_owner="nephe-ci"
export TF_VAR_region="us-west-1"

export AWS_ACCESS_KEY_ID=${TF_VAR_aws_access_key_id}
export AWS_SECRET_ACCESS_KEY=${TF_VAR_aws_access_key_secret}
export AWS_DEFAULT_REGION=${TF_VAR_region}
export TF_VAR_eks_key_pair_name=${KEY_PAIR}
export TF_VAR_eks_cluster_iam_role_name=k8s
export TF_VAR_eks_iam_instance_profile_name="suwang-eks-worker-nodes-1-NodeInstanceProfile-8LL239W4W3GN"

aws ec2 import-key-pair --key-name ${KEY_PAIR} --public-key-material fileb://~/.ssh/id_rsa.pub --region ${TF_VAR_region}


hack/install-cloud-tools.sh
echo "Creating EKS Cluster"
$HOME/terraform/eks create

aws ec2 delete-key-pair  --key-name ${KEY_PAIR}  --region ${TF_VAR_region}

echo "Destroying EKS Cluster"
$HOME/terraform/eks destroy

exit 0
echo "Installing Go 1.17"
curl -LO https://golang.org/dl/go1.17.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -zxf go1.17.linux-amd64.tar.gz -C /usr/local/
rm go1.17.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

echo "Building Nephe Docker image"
make build

echo "Pulling Docker images"
docker pull kennethreitz/httpbin
docker pull byrnedo/alpine-curl
docker pull quay.io/jetstack/cert-manager-controller:v1.8.2
docker pull quay.io/jetstack/cert-manager-webhook:v1.8.2
docker pull quay.io/jetstack/cert-manager-cainjector:v1.8.2
docker pull projects.registry.vmware.com/antrea/antrea-ubuntu:v1.8.0
