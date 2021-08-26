#  Part 1 - Managing Google Cloud Resources with Config Connector

![](/images/gcp.png)

## Install prerequisites

- gcloud SDK
- kubectl
- AWS CLI

## Create a GCP project

- Create a Google Cloud Project
- Set up billing for your project

```
export PROJECT_ID="<your-project-id>"
```

- Enable APIs

```
gcloud config set project $PROJECT_ID

gcloud services enable container.googleapis.com
gcloud services enable redis.googleapis.com
```

## Create a GKE Cluster

Create a GKE cluster with Workload Identity and Config Connector enabled.

```
gcloud container clusters create "cymbal-shops" --zone "us-central1-c" \
--release-channel "regular" --machine-type "e2-standard-4" --num-nodes "4" \
--addons HorizontalPodAutoscaling,HttpLoadBalancing,GcePersistentDiskCsiDriver,ConfigConnector \
--workload-pool "${PROJECT_ID}.svc.id.goog" --enable-stackdriver-kubernetes
```

## Install Config Connector to GKE

- Connect to the cluster:

```
gcloud container clusters get-credentials cymbal-shops --zone us-central1-c --project $PROJECT_ID
```

- Set up Config Connector's service account. This allows it to talk to GCP from inside the GKE cluster.

```
gcloud iam service-accounts create config-connector

gcloud projects add-iam-policy-binding ${PROJECT_ID} \
    --member="serviceAccount:config-connector@${PROJECT_ID}.iam.gserviceaccount.com" \
    --role="roles/owner"

gcloud iam service-accounts add-iam-policy-binding \
config-connector@${PROJECT_ID}.iam.gserviceaccount.com \
    --member="serviceAccount:${PROJECT_ID}.svc.id.goog[cnrm-system/cnrm-controller-manager]" \
    --role="roles/iam.workloadIdentityUser"
```

- Open `configconnector.yaml` in this directly. Replace `PROJECT_ID` with your project ID.


- Apply the install configuration to your cluster.

```
kubectl apply -f configconnector.yaml
```

- Determine what K8s namespace you'll deploy Config Connector resources into.

```
kubectl create namespace config-connector-resources

kubectl annotate namespace config-connector-resources cnrm.cloud.google.com/project-id=${PROJECT_ID}
```

- Verify your installation.

```
kubectl wait -n cnrm-system --for=condition=Ready pod --all
```

*Expected output*

```
pod/cnrm-controller-manager-0 condition met
pod/cnrm-deletiondefender-0 condition met
pod/cnrm-resource-stats-recorder-6b7bdb6bb4-9567n condition met
pod/cnrm-webhook-manager-567789d76c-pbhpk condition met
pod/cnrm-webhook-manager-567789d76c-thcj5 condition met
```

## Set up GKE workload identity with Config Connector

GKE Workload Identity is a mapping from a GCP IAM service account to a Kubernetes Service Account. You then assign IAM roles to your GCP service account which then apply to a K8s service account, mounted into a pod. This allows your Kubernetes workloads to have specific GCP permissions. In our case, we'll configure the GCP service account [from within Config Connector](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity#config-connector), then use that service account mapping to let the Cymbal Shops app (inside GKE) access Cloud Memorystore (in hosted GCP).

Create your Kubernetes service account:

```
kubectl create namespace cymbal-shops
kubectl create serviceaccount --namespace cymbal-shops cymbal-ksa
```

Create the GCP Service Account as YAML:

```
kubectl apply -f cymbal-gsa.yaml
```

Verify that the GCP service account gets created:

```
kubectl get gcp -n config-connector-resources
```

*Expected output*

```
NAME                                                     AGE   READY   STATUS     STATUS AGE
iamserviceaccount.iam.cnrm.cloud.google.com/cymbal-gsa   57s   True    UpToDate   57s
```


Map the GSA to the KSA using an IAMPolicyBinding defined as YAML:

```
kubectl apply -f workload-identity-binding.yaml
```

Verify that the IAMPolicyBinding is created:

```
kubectl get gcp -n config-connector-resources
```

*Expected output*

```
NAME                                                     AGE     READY   STATUS     STATUS AGE
iamserviceaccount.iam.cnrm.cloud.google.com/cymbal-gsa   2m21s   True    UpToDate   2m21s

NAME                                                                     AGE   READY   STATUS     STATUS AGE
iampolicy.iam.cnrm.cloud.google.com/iampolicy-workload-identity-sample   38s   True    UpToDate   37s
```

Finally, annotate the Kubernetes service account to map to the GSA.

```
kubectl annotate serviceaccount \
  --namespace cymbal-shops \
  cymbal-ksa \
  iam.gke.io/gcp-service-account=cymbal-gsa@${PROJECT_ID}.iam.gserviceaccount.com
```

## Use Config Connector to create a Cloud Memorystore Instance

Grant the `cymbal-gsa` Google service account `Cloud Memorystore` read-write permissions.

```
gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --member "serviceAccount:cymbal-gsa@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role roles/redis.editor
```

Use Config Connector to spin up a new [Cloud Memorystore Redis](https://cloud.google.com/config-connector/docs/reference/resource-docs/redis/redisinstance#sample_yamls) instance in your project.

```
kubectl apply -f memorystore-redis.yaml
```

Verify that the Memorystore instance gets created. **Note** - this usually takes 3-5 minutes.

```
kubectl get gcp
```

Get the Redis IP for your cloud-hosted instance.

```
gcloud redis instances list --region us-central1
```

*Expected output*:

```
INSTANCE_NAME  VERSION    REGION       TIER   SIZE_GB  HOST         PORT  NETWORK  RESERVED_IP     STATU
S  CREATE_TIME
redis-cart     REDIS_5_0  us-central1  BASIC  1        10.243.8.75  6379  default  10.243.8.72/29  READY
   2021-08-24T17:25:05
```

Update `cymbal-shops.yaml` with your Redis IP. (Line 414)

## Deploy the Cymbal Shops app to GKE

Apply the Kubernetes manifests to deploy the Cymbal Shops app to the GKE cluster. Note that these workloads use the `cymbal-ksa` service account, meaning that `cartservice`, which talks to the Cloud Memorystore redis instance, has the permissions it needs (via Workload Identity) to talk to GCP.

```
kubectl apply -n cymbal-shops -f cymbal-shops.yaml
```

Verify that pods are running.

Access the frontend.
