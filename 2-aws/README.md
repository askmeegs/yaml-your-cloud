# Part 2 - Managing AWS Resources with ACK 

![](/images/aws.png)

## Set up AWS local client 

- Install tools:
  - `aws-cli`
  - `eksctl` 
  - helm 3 

https://aws.github.io/aws-sdk-go-v2/docs/getting-started/ 


- Get an AWS access key and secret ID. 

- Download key to this location: `~/.aws/accesskey.csv`

- Manually create an S3 bucket in the AWS Console 

- Install the AWS Command line tool. https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2-mac.html \

- Configure the AWS command line tool with your access key and secret ID. For `region`, use `us-east-1`. For `default-output-format`, use `table`. 

```
aws configure
```

## Create EKS Cluster 

- Create an EC2 key pair. 

```
aws ec2 create-key-pair --region us-east-1 --key-name cymbal-eks-key
```

- Create an EKS cluster with a single EC2 NodeGroup. 

```
eksctl create cluster \
--name cymbal-eks \
--region us-east-1 \
--with-oidc \
--ssh-access \
--ssh-public-key cymbal-eks-key \
--managed
```

- Verify that the Kubernetes API is reachable 

```
kubectl get nodes
```

*Expected output* 

```
NAME                             STATUS   ROLES    AGE     VERSION
ip-192-168-58-194.ec2.internal   Ready    <none>   4m52s   v1.20.4-eks-6b7464
ip-192-168-7-20.ec2.internal     Ready    <none>   4m55s   v1.20.4-eks-6b7464
```

## Install ACK S3 Controller 

https://github.com/aws-controllers-k8s/s3-controller

https://aws-controllers-k8s.github.io/community/user-docs/install/ 

1. Download the helm chart. 

```
export HELM_EXPERIMENTAL_OCI=1
export SERVICE=s3
export RELEASE_VERSION=v0.0.1
export CHART_EXPORT_PATH=/tmp/chart
export CHART_REPO=public.ecr.aws/aws-controllers-k8s/$SERVICE-chart
export CHART_REF=$CHART_REPO:$RELEASE_VERSION

mkdir -p $CHART_EXPORT_PATH

helm chart pull $CHART_REF
helm chart export $CHART_REF --destination $CHART_EXPORT_PATH
```

1. Set up namespace

```
kubectl create namespace ack-system

```

3. Configure the S3 controller to use the `ack-s3-sa` service account. 

*Edit values.yaml in /tmp/chart with a Role ARN that can access S3* 

4. Install the Helm chart for the S3 controller to the ack-system namespace in EKS. 

```
helm install --namespace $ACK_K8S_NAMESPACE ack-$SERVICE-controller \
    -f $CHART_EXPORT_PATH/ack-$SERVICE-controller/values.yaml \
    $CHART_EXPORT_PATH/ack-$SERVICE-controller
```

## Create an S3 Bucket using ACK 

https://aws-controllers-k8s.github.io/community/reference/S3/v1alpha1/Bucket/

```
kubectl apply -f bucket.yaml
```

## Set up AWS service accounts for EKS 

(To allow Cymbal Ads, running in EKS, to access S3)

1. **Create an AWS IAM Policy for S3 access.** 

https://docs.aws.amazon.com/eks/latest/userguide/create-service-account-iam-policy-and-role.html 

2. **Create an AWS IAM role, using that S3 IAM policy.** 

```
kubectl create namespace cymbal-ads 

IAM_POLICY_ARN="<your-arn-here>"

eksctl create iamserviceaccount \
    --name cymbal-ads-air \
    --namespace cymbal-ads \
    --cluster cymbal-eks \
    --attach-policy-arn $IAM_POLICY_ARN \
    --approve \
    --override-existing-serviceaccounts
```

## Deploy the Cymbal Ads service to EKS 

Set the `BUCKET_NAME` env variable in `cymbal-ads.yaml`. 

Apply the service to the cluster. 

```
kubectl apply -f cymbal-ads.yaml 
```

Note - the Cymbal Ads deployment uses the Kubernetes service account you just set up, so the Pod can read/write from S3. 

Verify that the app is running: 

Access the cymbal ads page by getting the external IP of the service type=Loadbalancer. You should see an advertisement in your web browser.
