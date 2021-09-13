# Part 1 - AWS Controllers for Kubernetes

![](/images/aws.png)

## Prerequisites 

1. AWS account with Billing enabled.
2. AWS IAM user with Console access.


## Demo 

### 1. Install required tools. 

  - `aws`: https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html
  - `eksctl`: https://eksctl.io/
  - `kubectl`: https://kubernetes.io/docs/tasks/tools/#kubectl
  - `helm`: https://helm.sh/docs/intro/install/


### 2. Configure AWS CLI authentication.

- [Get an AWS access key and secret ID](https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html#access-keys-and-secret-access-keys). 
- Download key to this location: `~/.aws/accesskey.csv`
- Configure the AWS command line tool:

```bash
aws configure
```

When prompted for your Access Key and Secret ID, paste the values in your downloaded key. When prompted for `region`, use your desired region (eg. `us-east-1`). For `default-output-format`, use `table`. This makes it easy to view [ARNs](https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html) (resources). 

#### 3. Create an Elastic Kubernetes Service (EKS) Cluster 

- Create an AWS EC2 key pair. 

```bash
aws ec2 create-key-pair --region us-east-1 --key-name cymbal-eks-key
```

- Create an EKS cluster with a single EC2 NodeGroup. 

```bash
eksctl create cluster \
--name cymbal-eks \
--region us-east-1 \
--with-oidc \
--ssh-access \
--ssh-public-key cymbal-eks-key \
--managed
```

- Verify that the Kubernetes API is reachable. 

```
kubectl get nodes
```

*Expected output* 

```
NAME                             STATUS   ROLES    AGE     VERSION
ip-192-168-58-194.ec2.internal   Ready    <none>   4m52s   v1.20.4-eks-6b7464
ip-192-168-7-20.ec2.internal     Ready    <none>   4m55s   v1.20.4-eks-6b7464
```

### 4. Install the ACK S3 Controller 

- Read the basic documentation about ACK and the S3 controller before installing it onto your EKS cluster, especially if your cluster lives in a shared organization. (This controller will be able to create, update, and delete S3 buckets in your account.)

https://aws-controllers-k8s.github.io/community/user-docs/install/ 
https://github.com/aws-controllers-k8s/s3-controller

- Download the ACK S3 Helm chart. 

```
export HELM_EXPERIMENTAL_OCI=1
export SERVICE=s3
export RELEASE_VERSION=v0.0.3
export CHART_EXPORT_PATH=/tmp/chart/v0.0.3
export CHART_REPO=public.ecr.aws/aws-controllers-k8s/$SERVICE-chart
export CHART_REF=$CHART_REPO:$RELEASE_VERSION

mkdir -p $CHART_EXPORT_PATH

helm chart pull $CHART_REF
helm chart export $CHART_REF --destination $CHART_EXPORT_PATH
```

- Set up the EKS namespace that the ACK controller will be installed into. 

```
kubectl create namespace ack-system
```

- Create an [AWS IAMPolicy](https://docs.aws.amazon.com/eks/latest/userguide/create-service-account-iam-policy-and-role.html) that has specific permissions to view and edit S3 buckets. Copy the Policy ARN (resource ID) to the clipboard. 

- Open `/tmp/chart/v0.0.3/s3-chart/values.yaml`. At the bottom of the file, uncomment the policy ARN annotation, and set the value to your S3 IAM policy. 

```yaml
serviceAccount:
  create: true
  name: ack-s3-controller
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::[YOUR_AWS_ACCOUNT_ID]:role/[YOUR_IAM_ROLE_NAME]
```

- Install the Helm chart to your EKS cluster. 
  
```
helm install --namespace ack-system ack-s3-controller \
-f /tmp/chart/v0.0.3/s3-chart/values.yaml \
/tmp/chart/v0.0.3/s3-chart
```

### 5 - Create an S3 Bucket using ACK 

*Reference* - https://aws-controllers-k8s.github.io/community/reference/S3/v1alpha1/Bucket/

```
kubectl apply -f bucket.yaml
```

Navigate to the AWS S3 Dashboard. You should see a Bucket named `cymbal-test`.  

### 6 - Set up the Cymbal Ads service. 

Cymbal Ads is a Kubernetes workload that serves a simple HTML advertisement in the browser. On startup, this workload writes the HTML advertisement to an S3 bucket. Then when a user requests the cymbal ads server, cymbal ads downloads the ad from the S3 bucket and serves it in the response. 

- Create the cymbal ads namespace in your EKS cluster. 

```
kubectl create namespace cymbal-ads 
```


- Copy your S3 IAM Policy ARN to the clipboard again. 
- Create an EKS service account, referencing that S3 IAM policy:  

```bash
IAM_POLICY_ARN="<your-arn-here>"

eksctl create iamserviceaccount \
    --name cymbal-ads-air \
    --namespace cymbal-ads \
    --cluster cymbal-eks \
    --attach-policy-arn $IAM_POLICY_ARN \
    --approve \
    --override-existing-serviceaccounts
```

### 7 - Deploy the Cymbal Ads service to EKS.

```
kubectl apply -f cymbal-ads.yaml 
```

This file contains a Kubernetes Deployment which references your newly-created S3 bucket, `cymbal-test`.  

You can reach the Cymbal Ads UI by running: `kubectl get svc -n cymbal-ads` and navigating to the `EXTERNAL_IP` in a browser. 
