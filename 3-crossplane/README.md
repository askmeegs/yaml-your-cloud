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

```sh
kubectl apply -f gcp/gcp_provider.yaml
kubectl apply -f aws/aws_provider.yaml
```

#### Wait for the providers to be ready
```sh
kubectl get provider

NAME           INSTALLED   HEALTHY   PACKAGE                         AGE
provider-aws   True        True      crossplane/provider-aws:alpha   27m
provider-gcp   True        True      crossplane/provider-gcp:alpha   27m
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

## Deploy Crossplane resources for Memorystore and S3
#### Deploy a resources of types `CloudMemorystoreInstance` and `Bucket` to your cluster
```sh
kubectl apply -f gcp/memory_store.yaml
kubectl apply -f aws/s3_bucket.yaml
```
#### Watch for changes and wait until the resources are ready
```sh
kubectl get cloudmemorystoreinstance.cache.gcp.crossplane.io/cymbal-memstore

NAME              READY   SYNCED   STATE   VERSION     AGE
cymbal-memstore   True    True     READY   REDIS_4_0   16m

kubectl get bucket.s3.aws.crossplane.io/cymbal-bucket

NAME            READY   SYNCED   AGE
cymbal-bucket   True    True     31s
```
## Configure access control

### Google Cloud Platform

#### Create namespace where the workload will be deployed
```sh
# this might already exist from the first demo
kubectl create namespace cymbal-shops
```

#### Setup [workload-identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity#gcloud) to enable access to the memorystore from the workload
```sh
# create a kubernetes service account 'cymbal-ksa'
# this might already exist from the first demo
kubectl create serviceaccount --namespace cymbal-shops cymbal-ksa

# create a GCP IAM service account
gcloud iam service-accounts create workload-id-sa

# add IAM policy binding to enable editor access to memorystore
gcloud projects add-iam-policy-binding yaml-shabirs-cloud \
  --member "serviceAccount:workload-id-sa@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role "roles/redis.editor"

# create an IAM policy binding between the kubernetes service account and
# the GCP IAM service account
gcloud iam service-accounts add-iam-policy-binding \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:${PROJECT_ID}.svc.id.goog[cymbal-shops/cymbal-ksa]" \
  workload-id-sa@${PROJECT_ID}.iam.gserviceaccount.com

# annotate the kubernetes service account the IAM service account email
kubectl annotate serviceaccount \
  --namespace cymbal-shops cymbal-ksa \
  iam.gke.io/gcp-service-account=workload-id-sa@${PROJECT_ID}.iam.gserviceaccount.com
```

### Amazon Web Services Platform



## Verify App functionality

### Cymbal App running on GCP

#### Get the host IP of the memorystore created by Crossplane
```sh
gcloud redis instances list --region us-central1
```

#### Update cymbal-shops.yaml with your memorystore IP (Line 414) and deploy
```sh
kubectl apply -n cymbal-shops -f ../1-gcp/cymbal-shops.yaml
```

#### Verify that the pods are running and access the frontend
```sh
kubectl get services/frontend-external -n cymbal-shops

NAME                TYPE           CLUSTER-IP     EXTERNAL-IP    PORT(S)        AGE
frontend-external   LoadBalancer   10.96.15.201   34.70.143.34   80:31584/TCP   47m
```
