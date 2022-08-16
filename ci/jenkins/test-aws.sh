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

KIND_VERSION=v0.12.0
KUBECTL_VERSION=v1.24.1
TERRAFORM_VERSION=0.13.5

echo ${PWD}
ls ${PWD}
cd testrepo

echo "Building Nephe Docker image"
make build

echo "Pull Docker images"
docker pull kennethreitz/httpbin
docker pull byrnedo/alpine-curl
docker pull quay.io/jetstack/cert-manager-controller:v1.8.2
docker pull quay.io/jetstack/cert-manager-webhook:v1.8.2
docker pull quay.io/jetstack/cert-manager-cainjector:v1.8.2
docker pull projects.registry.vmware.com/antrea/antrea-ubuntu:v1.7.0

echo "Create kind cluster"
hack/install-cloud-tools.sh
ci/kind/kind-setup.sh create kind

echo "this is a test script, username $AWS_KEY_PAIR_NAME" > test.log
