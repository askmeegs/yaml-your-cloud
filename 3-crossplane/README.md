# Part 3 - Managing Multi-cloud resources with Crossplane

## Pre-requisites
- helm
- kubectl _(context configure to point to the cluster of your choice)_
## Remove Config Connector and ACK


## Install Crossplane

#### Setup the environment variables
```sh
PROJECT_ID=<YOUR_GCP_PROJECT>
NEW_SA_NAME=<GCP_SERVICE_ACCOUNT> # give any name of your choice
NAMESPACE=<CROSSPLANE_NAMESPACE> # name to install Crossplane components
```

#### Download and install the Crossplane CLI
```sh
# this will run the installation script and print some 'echo' commands; you should run them
curl -sL https://raw.githubusercontent.com/crossplane/crossplane/master/install.sh | sh
```

#### Create the namespace for the Crossplane components
```sh
kubectl create namespace ${NAMESPACE}
```

#### Download the `helm charts` for the Crossplane CRDs and install using `helm`
```
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm repo update
helm install crossplane --namespace $NAMESPACE crossplane-stable/crossplane --version 1.3.1
```

#### Verify that the components have been installed properly
```sh
helm list -n ${NAMESPACE}
kubectl get all -n ${NAMESPACE}
```

## Install cloud resource specific Crossplane configurations into the cluster

#### Install the GCP provider
```sh
kubectl apply -f gcp/gcp_provider.yaml
kubectl apply -f aws/aws_provider.yaml
```

## Configure Crossplane provider in the cluster

#### GCP
_(you should have gcloud configured to the correct GCP project)_
```sh
# create GCP service account
SA="${NEW_SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"
gcloud iam service-accounts create $NEW_SA_NAME --project ${PROJECT_ID}

# enable required APIs
gcloud services enable redis.googleapis.com --project $PROJECT_ID

# grant the service-account access to the cloud API and download service-account key
```sh
ROLE="roles/redis.admin"
gcloud projects add-iam-policy-binding --role="$ROLE" ${PROJECT_ID} --member "serviceAccount:$SA"

# create service account key file
gcloud iam service-accounts keys create gcp-creds.json --project ${PROJECT_ID} --iam-account $SA
```

#### AWS
_(you should have the aws CLI configured to the correct AWS profile)_
```sh
AWS_PROFILE=default && echo -e "[default]\naws_access_key_id = $(aws configure get aws_access_key_id --profile $AWS_PROFILE)\naws_secret_access_key = $(aws configure get aws_secret_access_key --profile $AWS_PROFILE)" > aws-creds.conf
```

#### Create kubenetes secret to serve as the provider secret
```sh
# for GCP provider
kubectl create secret generic gcp-creds -n ${NAMESPACE} --from-file=creds=./gcp-creds.json
# for AWS provider
kubectl create secret generic aws-creds -n ${NAMESPACE} --from-file=creds=./aws-creds.conf
```

#### Setup configurations for the Crossplane providers (GCP & AWS)
```sh
# for GCP provider
echo "apiVersion: gcp.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: gcp-config
spec:
  projectID: ${PROJECT_ID}
  credentials:
    source: Secret
    secretRef:
      namespace: ${NAMESPACE}
      name: gcp-creds
      key: creds" | kubectl apply -f -

# for AWS provider
echo "apiVersion: aws.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: aws-config
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: ${NAMESPACE}
      name: aws-creds
      key: creds" | kubectl apply -f -
```

## Deploy Crossplane resources for Memorystore, IAM, and S3
#### Deploy a resources of types `CloudMemorystoreInstance` and `Bucket` to your cluster
```sh
kubectl apply -f gcp/memory_store.yaml
kubectl apply -f aws/s3_bucket.yaml
```

#### Watch for changes and wait until the resources are ready
```sh
kubectl get cloudmemorystoreinstance cymbal-memstore

NAME              READY   SYNCED   STATE   VERSION     AGE
cymbal-memstore   True    True     READY   REDIS_4_0   4m37s
```
## Verify Cymbal app functionality
