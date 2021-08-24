AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query "Account" --output text)
EKS_CLUSTER_NAME="cymbal-eks"
OIDC_PROVIDER=$(aws eks describe-cluster --name $EKS_CLUSTER_NAME --query "cluster.identity.oidc.issuer" --output text | sed -e "s/^https:\/\///")
ACK_K8S_NAMESPACE=ack-system
ACK_K8S_SERVICE_ACCOUNT_NAME=ack-s3-controller

read -r -d '' TRUST_RELATIONSHIP <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::${AWS_ACCOUNT_ID}:oidc-provider/${OIDC_PROVIDER}"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "${OIDC_PROVIDER}:sub": "system:serviceaccount:${ACK_K8S_NAMESPACE}:${ACK_K8S_SERVICE_ACCOUNT_NAME}"
        }
      }
    }
  ]
}
EOF
echo "${TRUST_RELATIONSHIP}" > trust.json

# update variables as needed
AWS_SERVICE_NAME='s3'
ACK_CONTROLLER_IAM_ROLE="ack-${AWS_SERVICE_NAME}-controller"
ACK_CONTROLLER_IAM_ROLE_DESCRIPTION='IRSA role for ACK $AWS_SERVICE_NAME controller deployment on EKS cluster using Helm charts'
aws iam create-role --role-name "${ACK_CONTROLLER_IAM_ROLE}" --assume-role-policy-document file://trust.json --description "${ACK_CONTROLLER_IAM_ROLE_DESCRIPTION}"
