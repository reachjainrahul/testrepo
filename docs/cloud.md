# cloud provider Cluster

## Common Pre-requisites
1. To run EKS cluster, install and configure aws cli, see
   https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html, and
   https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html
1. Install aws-iam-authenticator, 
   https://docs.aws.amazon.com/eks/latest/userguide/install-aws-iam-authenticator.html
1. To run AKS cluster, install and configure azure cli,
   https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest
1. Install terraform
   see https://learn.hashicorp.com/terraform/getting-started/install.html
1. Add the following environment variable in your local dev so that you can identify
   your own cloud resources.```export TF_VAR_owner=YOUR_NAME```, Where TF_VAR_owner should be one
   word (contain no spaces) and should be in lower case.
1. You must already have ssh key-pair created. This key pair will be used to access worker node via
   ssh. ([This link](https://awscli.amazonaws.com/v2/documentation/api/latest/reference/ec2/import-key-pair.html) shows how to create a Key Pair and configure to use
   it to access your EC2 worker node instances)
1. You need to have access to jfrog repo `antrea-service-tilt-images` and follow the below steps for auto pulling pod images.
   Otherwise, you'll need to upload antreaplus and other images manually.
   ```bash
   export TF_VAR_jfrog_usr=YOUR_JFROG_USER_NAME
   export TF_VAR_jfrog_api_key=YOUR_JFROG_API_KEY
   ```
   Where
   - TF_VAR_jfrog_usr should be your jfrog username, same as vmware username
   - TF_VAR_jfrog_api_key should be your jfrog api key, created [here](https://vmwaresaas.jfrog.io/ui/admin/artifactory/user_profile)

NOTE: For version of third-party tools used refer to cloudcontroller/.gitlab-ci.yaml file

## Create an EKS cluster via terraform
Ensures that you have permission to create EKS cluster, and have already
created EKS cluster role as well as worker node profile.
```bash
export TF_VAR_eks_cluster_iam_role_name=YOUR_EKS_ROLE
export TF_VAR_eks_iam_instance_profile_name=YOUR_EKS_WORKER_NODE_PROFILE
export TF_VAR_eks_key_pair_name=YOUR_KEY_PAIR_TO_ACCESS_WORKER_NODE
```
Where 
- TF_VAR_eks_cluster_iam_role_name may be created [here](https://docs.aws.amazon.com/eks/latest/userguide/service_IAM_role.html#create-service-role)
    * For our testing we share the role, please ask
- TF_VAR_eks_iam_instance_profile_name may be created [here](
https://docs.aws.amazon.com/eks/latest/userguide/worker_node_IAM_role.html#create-worker-node-role)
    * For our testing we share the worker profile name, please ask
- TF_VAR_eks_key_pair_name may be configured [here](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html#how-to-generate-your-own-key-and-import-it-to-aws)
and it should be copied from ``~/.ssh/id_rsa``.

Install cloud tools
```bash
 cloudcontroller$ ./hack/install-cloud-tools.sh
```
Create EKS cluster
```bash
~/terraform/eks create
```
Interact with EKS cluster
```
~/terraform/eks kubectl ... // issue kubectl commands to EKS cluster
~/terraform/eks load ... // load local built images to EKS cluster
~/terraform/eks destroy // destroy EKS cluster
~/terraform/eks output  // display EKS attributes
```
once EKS is created, worker nodes are accessible via their external IP using ssh.

Load Antrea plus image to EKS worker nodes
```bash
~/terraform/eks load antreaplus/antreaplus
```

Deploy Antrea plus
```bash
~/terraform/eks kubectl apply -f config/antrea-plus.yaml
```
## Create an AKS cluster via terraform
Create or obtain a azure service principal
[here](https://docs.microsoft.com/en-us/azure/active-directory/develop/howto-create-service-principal-portal#create-an-azure-active-directory-application). And set the following environment variables from
 the service principal. For instance in .bashrc
```bash
export TF_VAR_aks_client_id=YOUR_SERVICE_PRINCIPAL_ID
export TF_VAR_aks_client_secret=YOUR_SERVICE_PRINCIPAL_SECRET
export TF_VAR_aks_client_subscription_id=YOUR_SUBCRIPTION_ID
export TF_VAR_aks_client_tenant_id=YOUR_TENANT_ID
```
[OPTIONAL] Enable pod identity role based access for AKS
```bash
export TF_VAR_cloud_controller_identity_id=MANAGED_IDENTITY_RESOURCE_ID
```
Install cloud tools
```bash
 ./hack/install-cloud-tools.sh
```
Create AKS cluster
```bash
~/terraform/aks create
```
Interact with AKS cluster
```
~/terraform/aks kubectl ... // issue kubectl commands to AKS cluster
~/terraform/aks load ... // load local built images to AKS cluster
~/terraform/aks destroy // destroy AKS cluster 
```
and worker nodes are accessible via their external IP using ssh.

## Add Antrea File Server
Antrea File Server will be hosting VM Agent bits. It is required if any VPC/VNET needs to be managed agented.

Load antrea file server image to worker nodes. <prefix> is ~/terraform/eks or ~/terraform/aks
```bash
<prefix> load antreaplus/fileserver
```

Deploy antrea file server secrets. The yaml manifest contains tls.crt tls.key and apikey and they can be generated as
```bash
# Create a public private key pair
$ openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout /tmp/tls.key -out /tmp/tls.crt -subj "/CN=antrea-file-server.kube-system.svc"
# Convert the keys to base64 encoding
$ cat /tmp/tls.crt | base64 | tr -d '\n'
$ cat /tmp/tls.key | base64 | tr -d '\n'

# Generate api key
$ openssl rand -base64 18
7B5zIqmRGXmrJTFmKa99vcit

# Deploy the yaml after updating certs and keys
<prefix> kubectl apply -f config/fileserver/antrea-file-server-secret.yaml
```

Deploy file server. It will create 2 replicas of antrea file server
```bash
<prefix> kubectl apply -f config/fileserver/antrea-file-server.yaml
```

## Create an AWS VM cluster
With the steps below you can create an agentless EC2 Cluster of 3 VMs on which apache tomcat server is deployed on port 80.
Each VM will have a public IP, and you can curl PUBLIC_IP:80 to access a sample web page.
Terraform variables can be modified to deploy agented cluster etc.

Create or obtain AWS key and secret, and configure environment variables. For instance in .bashrc

```bash
export TF_VAR_region=YOUR_REGION
export TF_VAR_owner=YOUR_ID
export TF_VAR_aws_access_key_id=YOUR_AWS_KEY
export TF_VAR_aws_access_key_secret=YOUR_AWS_KEY_SECRET
export TF_VAR_aws_key_pair_name=YOU_AWS_KEY_PAIR
```
NOTE: Choose a region closest to you for a faster deployment.

Install cloud tools
```bash
 ./hack/install-cloud-tools.sh
```
Create AWS VM cluster
```bash
~/terraform/aws-tf create
```

Interact with AWS VM cluster
```
~/terraform/aws-tf output // AWS VPC attributes 
~/terraform/aws-tf destroy // destroy AWS VPC cluster
```

Create/Destroy VM cluster Peered with EKS cluster
```bash
TF_VAR_peer_vpc_id=EKS_VPC_ID ~/terraform/aws-tf create peer
TF_VAR_peer_vpc_id=EKS_VPC_ID ~/terraform/aws-tf destroy peer
```
where you may get EKS_VPC_ID from
```bash
~/terraform/eks output
```

You can also create another vpc (we call it peer_vpc) and a peer connection between peer_vpc and the first vpc. Interaction with peer_vpc are also similar.
```bash
~/terraform/aws-tf create peer
~/terraform/aws-tf output peer
~/terraform/aws-tf output peer vpc_id
~/terraform/aws-tf destroy peer
```

For destroying, you have to destroy the peer connection and peer_vpc first. So the overall sequence of creating and destroying will be:
```bash
~/terraform/aws-tf create
~/terraform/aws-tf create peer
~/terraform/aws-tf destroy peer
~/terraform/aws-tf destroy
```

If Create/Destroy was interrupted and fails due to partially cleaned state

1. Manually delete the VM instances and VPC from amazon console
2. rm -rf ~/tmp/terraform-aws/ to clear stale terraform state(rm -rf ~/tmp2/terraform-aws/ for peer_vpc).
3. And re-create

## Create an Azure VM cluster
Create or obtain Azure Service Principal credential, and configure environment variables. For
instance in .bashrc

```bash
export TF_VAR_azure_client_id=YOUR_CLIENT_ID
export TF_VAR_azure_client_secret=YOUR_CLIENT_SECRET
export TF_VAR_azure_client_subscription_id=YOUR_SUBCRIPTION_ID
export TF_VAR_azure_client_tenant_id=YOUR_TENANT_ID
```
Install cloud tools
```bash
 ./hack/install-cloud-tools.sh
```
Create Azure VM cluster
```bash
~/terraform/azure-tf create
```

Interact with Azure VM cluster
```
~/terraform/azure-tf output // Azure VNet attributes 
~/terraform/azure-tf destroy // destroy Azure VNet cluster
```

You can also create another resource group (it has a vnet and we call it peer_vnet) and two peerings between peer_vnet and the first one. Interaction with the second resource group are also similar.
```bash
~/terraform/azure-tf create peer
~/terraform/azure-tf output peer
~/terraform/azure-tf output peer vnet_id
~/terraform/azure-tf destroy peer
```

For destroying, the sequence does not matter.
