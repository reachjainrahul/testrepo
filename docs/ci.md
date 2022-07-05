# CI Pipeline

## Tests
CI Pipeline consists of two types for tests.
* Core: These are the tests that always run at merge request, merge to master and nightly. They
 include unit-test, vm-agent-test, integration-test.
* Extended: These are the tests, due to time, resource or resource constraints, are not run at
 merge request, merge to master, but will run nightly. They include
  * agent-eks
  * agent-aks 
  * azure-agentless
  * federation

The extended tests may be trigger to run in merge request or/and merge to master by including in
commit message `extended-test-all`, this will run all extended tests. A specific extended test
may also be triggered by providing its name in commit message, such as `extended-test-agent-eks`.

## Variables

These are the environment variables needed in CI/CD pipeline for the tests.

### Core tests
* AWS_SSH_KEY: The private key of TF_VAR_aws_key_pair_name that is used by test to ssh into VMs.
* DOCKER_AUTH_CONFIG: Docker configuration used to pull images from harbor repo. Please remove the “credsStore” key pair if you generate it automatically, it will cause error when running the aks integration test.
* TF_VAR_aws_access_key_id: AWS user key.
* TF_VAR_aws_access_key_secret: AWS user secret.
* TF_VAR_aws_key_pair_name: The name of an existing Aws key pair, which has private key specified
 by AWS_SSH_KEY.
 
### Extended Azure Agentless tests
* AZURE_SSH_PRIV_KEY: Private ssh key usd by test to ssh into VMs.
* AZURE_SSH_PUB_KEY: Public ssh key that will be copied onto VMs.
* DOCKER_AUTH_CONFIG: Docker configuration used to pull images from harbor repo.
* TF_VAR_azure_client_id: Azure application/client ID.
* TF_VAR_azure_client_secret: Azure client secret.
* TF_VAR_azure_client_subscription_id: Azure client subscription ID.
* TF_VAR_azure_client_tenant_id: Azure client tenant ID.

### Extended agent-eks
* AGENT_DOCKER_USER: User from dockerhub used by VM to retrieve agent from dockerhub.
* AGENT_DOCKER_PWD:  Passwd from dockerhub used by VM to retrieve agent from dockerhub.
* TF_VAR_eks_cluster_iam_role_name: An existing role that allows EKS creation.
* TF_VAR_eks_iam_instance_profile_name: an existing profile that assigned to EKS worker node.
* TF_VAR_eks_key_pair_name: The name of an existing Aws key pair, which has private key specified
by AWS_SSH_KEY.
* [OPTIONAL] TF_VAR_cloud_controller_role_arn: The arn of the role used for cloud controller's role based access

### Extended agent-aks
* AGENT_DOCKER_USER: User from dockerhub used by VM to retrieve agent from dockerhub.
* AGENT_DOCKER_PWD:  Passwd from dockerhub used by VM to retrieve agent from dockerhub.
* AZURE_SSH_PRIV_KEY: Private ssh key usd by test to ssh into VMs.
* AZURE_SSH_PUB_KEY: Public ssh key that will be copied onto VMs.
* TF_VAR_azure_client_id: Azure application/client ID.
* TF_VAR_azure_client_secret: Azure client secret.
* TF_VAR_azure_client_subscription_id: Azure client subscription ID.
* TF_VAR_azure_client_tenant_id: Azure client tenant ID.
* TF_VAR_aks_client_id: Azure application/client ID.
* TF_VAR_aks_client_secret: Azure client secret.
* TF_VAR_aks_client_subscription_id: Azure client subscription ID.
* TF_VAR_aks_client_tenant_id: Azure client tenant ID.
* [OPTIONAL] TF_VAR_cloud_controller_identity_client_id: The client id of the identity used for cloud controller's role based access
