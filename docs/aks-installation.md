# Deploying Nephe in Azure AKS

## Prerequisites

1. Install and configure [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest).
2. Install [Terraform](https://learn.hashicorp.com/terraform/getting-started/install.html).
3. Install `jq`, `pv`, and `bzip2`.
4. Create or obtain an azure service principal and set the below environment
   variables. Please refer to [Azure documentation](https://docs.microsoft.com/en-us/azure/active-directory/develop/howto-create-service-principal-portal#create-an-azure-active-directory-application)
   for more information.

```bash
export TF_VAR_owner=YOUR_NAME
export TF_VAR_aks_client_id=YOUR_SERVICE_PRINCIPAL_ID
export TF_VAR_aks_client_secret=YOUR_SERVICE_PRINCIPAL_SECRET
export TF_VAR_aks_client_subscription_id=YOUR_SUBCRIPTION_ID
export TF_VAR_aks_client_tenant_id=YOUR_TENANT_ID
```

- `TF_VAR_owner` may be set so that you can identify your own cloud resources.
  It should be one word, with no spaces and in lower case.

## Create an AKS cluster via terraform

### Setup Terraform Environment

```bash
./hack/install-cloud-tools.sh
```

The [install cloud tools](../hack/install-cloud-tools.sh) script copies the
required bash and terraform scripts to the user home directory, under
`~/terraform/`.

### Create an AKS cluster

Create an AKS cluster using the provided terraform scripts. Once the AKS cluster
is created, worker nodes are accessible via their external IP using ssh.
Terraform state files and other runtime info will be stored under
`~/tmp/terraform-aks/`. You can also create an AKS cluster in other ways and
deploy prerequisites manually.

This will deploy `cert-manager v1.8.2` and `antrea v1.8`.

```bash
~/terraform/aks create
```

### Deploy Nephe Controller

```bash
~/terraform/aks kubectl apply -f config/nephe.yml
```

### Interact with AKS cluster

Issue kubectl commands to AKS cluster using the helper scripts. To run kubectl
commands directly, export `KUBECONFIG` environment variable.

```bash
~/terraform/aks kubectl ...
export KUBECONFIG=~/tmp/terraform-aks/kubeconfig
```

Loading locally built `antrea/nephe` image to AKS cluster.

```bash
~/terraform/aks load ...
~/terraform/aks load antrea/nephe
```
```

Display AKS attributes.

```bash
~/terraform/aks output
```

### Destroy AKS cluster

```bash
~/terraform/aks destroy
```

## Create Azure VMs

Additionally, you can also create a compute VNET with 3 VMs using the terraform
scripts for testing purpose. Each VM will have a public IP and an Apache Tomcat
server deployed on port 80. Use curl `<PUBLIC_IP>:80` to access a sample web
page. Create or obtain Azure Service Principal credential and configure the
below environment variables.

```bash
export TF_VAR_azure_client_id=YOUR_CLIENT_ID
export TF_VAR_azure_client_secret=YOUR_CLIENT_SECRET
export TF_VAR_azure_client_subscription_id=YOUR_SUBCRIPTION_ID
export TF_VAR_azure_client_tenant_id=YOUR_TENANT_ID
```

### Setup Terraform Environment

```bash
./hack/install-cloud-tools.sh
```

### Create VMs

```bash
~/terraform/azure-tf create
```

Terraform state files and other runtime info will be stored under
`~/tmp/terraform-azure/`

### Get VNET attributes

```bash
~/terraform/azure-tf output
```

### Destroy VMs

```bash
~/terraform/azure-tf destroy
```
