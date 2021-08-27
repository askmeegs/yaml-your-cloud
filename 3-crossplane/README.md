# Part 3 - Managing Multi-cloud resources with Crossplane

## Pre-requisites
- helm
- kubectl _(context configure to point to the cluster of your choice)_
## Remove Config Connector and ACK


## Install Crossplane

- Setup the environment variables
```sh
PROJECT_ID=<YOUR_GCP_PROJECT>
NEW_SA_NAME=<GCP_SERVICE_ACCOUNT> # give any name of your choice
NAMESPACE=<CROSSPLANE_NAMESPACE> # name to install Crossplane components
```

- Download and install the Crossplane CLI
```sh
# this will run the installtion script and print some 'echo' commands; you should run them
curl -sL https://raw.githubusercontent.com/crossplane/crossplane/master/install.sh | sh
```

- Create the namespace for the Crossplane components
```sh
kubectl create namespace ${NAMESPACE}
```

- Download the `helm charts` for the Crossplane CRDs and install using `helm`
```
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm repo update
helm install crossplane --namespace $NAMESPACE crossplane-stable/crossplane --version 1.3.1
```

- Verify that the components have been installed properly
```sh
helm list -n ${NAMESPACE}
kubectl get all -n ${NAMESPACE}
```

## Install cloud resource specific Crossplane configurations into the cluster

- Install the provider configuration
```sh
kubectl apply -f crossplane.yaml
```

## Configure Crossplane provider in the cluster
- Create GCP service account
```sh
SA="${NEW_SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"
gcloud iam service-accounts create $NEW_SA_NAME --project ${PROJECT_ID}
```

- Enable required APIs
```
gcloud services enable redis.googleapis.com --project $PROJECT_ID
```

- Grant the service-account access to the cloud API and download service-account key
```sh
ROLE="roles/redis.admin"
gcloud projects add-iam-policy-binding --role="$ROLE" ${PROJECT_ID} --member "serviceAccount:$SA"
```

- Create service account key file
```sh
gcloud iam service-accounts keys create creds.json --project ${PROJECT_ID} --iam-account $SA
```

- Create kubenetes secret to serve as the provider secret
```sh
kubectl create secret generic gcp-creds -n ${NAMESPACE} --from-file=creds=./creds.json
```

- Create `ProviderConfig` for the Crossplane cloud provider
```sh
echo "apiVersion: gcp.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  projectID: ${PROJECT_ID}
  credentials:
    source: Secret
    secretRef:
      namespace: ${NAMESPACE}
      name: gcp-creds
      key: creds" | kubectl apply -f -
```

## Deploy Crossplane resources for Memorystore, IAM, and S3
- Deploy a resource of type `CloudMemorystoreInstance` to your cluster
```sh
echo "apiVersion: cache.gcp.crossplane.io/v1beta1
kind: CloudMemorystoreInstance
metadata:
  name: cymbal-memstore
  namespace: default
spec:
  forProvider:
    memorySizeGb: 50
    region: us-central1
    tier: BASIC" | kubectl apply -f -
```

- Watch for changes and wait until the Postgres instance is ready
```sh
kubectl get cloudmemorystoreinstance cymbal-memstore

NAME              READY   SYNCED   STATE   VERSION     AGE
cymbal-memstore   True    True     READY   REDIS_4_0   4m37s
```
## Verify Cymbal app functionality
