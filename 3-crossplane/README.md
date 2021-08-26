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

- Replace the `NAMESPACE` in the `composition.yaml` file
```sh
mkdir crossplane-config
cd crossplane-config
sed -i -e "s/NAMESPACE/${NAMESPACE}/g" composition.yaml
```

- Build and push the configuration to `Google Container Registry`
```sh
# first enable the gcr API
gcloud services enable containerregistry.googleapis.com
# build the Crossplane configuration
kubectl crossplane build configuration
# finally push it to gcr
kubectl crossplane push configuration gcr.io/$PROJECT_ID/getting-started-with-gcp:v1.3.1
```

- Install the Crossplane configuration from the pushed image
```sh
kubectl crossplane install configuration gcr.io/$PROJECT_ID/getting-started-with-gcp:v1.3.1
```

- Validate the installation
```sh
# check until you see both `INSTALLED` and `READY` are True
kubectl get pkg

NAME                                                                       INSTALLED   HEALTHY   PACKAGE                                                  AGE
configuration.pkg.crossplane.io/yaml-your-cloud-getting-started-with-gcp   True        True      gcr.io/yaml-your-cloud/getting-started-with-gcp:v1.3.1   30m

NAME                                                 INSTALLED   HEALTHY   PACKAGE                           AGE
provider.pkg.crossplane.io/crossplane-provider-gcp   True        True      crossplane/provider-gcp:v0.17.1   30m
```

## Configure Crossplane provider in the cluster
- Create GCP service account
```sh
SA="${NEW_SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"
gcloud iam service-accounts create $NEW_SA_NAME --project $PROJECT_ID
```

- Enable required APIs
```
gcloud services enable sqladmin.googleapis.com --project $PROJECT_ID
```

- Grant the service-account access to the cloud API and download service-account key
```sh
ROLE="roles/cloudsql.admin"
gcloud projects add-iam-policy-binding --role="$ROLE" $PROJECT_ID --member "serviceAccount:$SA"
```

- Create service account key file
```sh
gcloud iam service-accounts keys create creds.json --project $PROJECT_ID --iam-account $SA
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
- Deploy a resource of type `PostgreSQLInstance` to your cluster
```sh
echo "apiVersion: database.example.org/v1alpha1
kind: PostgreSQLInstance
metadata:
  name: my-db
  namespace: default
spec:
  parameters:
    storageGB: 20
  compositionSelector:
    matchLabels:
      provider: gcp
  writeConnectionSecretToRef:
    name: db-conn" | kubectl apply -f -
```

- Watch for changes and wait until the Postgres instance is ready
```sh
kubectl get postgresqlinstance my-db
kubectl get crossplane -l crossplane.io/claim-name=my-db
```
## Verify Cymbal app functionality
