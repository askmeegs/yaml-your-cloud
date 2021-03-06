# Part 3 - Using OpenPolicyAgent for Hosted Resources

**Note** - this demo only shows OPA gatekeeper on the GKE (GCP) cluster.

#### 1. Install OPA Gatekeeper 3.5 to GKE 

https://open-policy-agent.github.io/gatekeeper/website/docs/install/

```
kubectl apply -f https://raw.githubusercontent.com/open-policy-agent/gatekeeper/release-3.5/deploy/gatekeeper.yaml
```

#### 2. Create a custom constraint template (Redis Version)

- https://github.com/open-policy-agent/gatekeeper/tree/master/demo/basic 
- https://cloud.google.com/sdk/gcloud/reference/beta/redis/instances/create#--redis-version 

This template will create logic to restrict Google Cloud Memorystore instances to a specific Redis version. 

```
kubectl apply -f template.yaml 
```

#### 3. Create a concrete constraint using the template. 

This will set a specific required version for Redis. Our constraint will require that any Google Cloud Memorystore instance *must* use Redis 5. 

```
kubectl apply -f constraint.yaml
```

#### 4. Attempt to apply a Memorystore instance that violates the RedisVersion policy. 

```
kubectl apply -f bad.yaml 
```

*Expected output* 

```
Error from server ([redis-require-v5] Redis Version REDIS_6_0 is not allowed, must use version: REDIS_5_0): error when creating "bad.yaml": admission webhook "validation.gatekeeper.sh" denied the request: [redis-require-v5] Redis Version REDIS_6_0 is not allowed, must use version: REDIS_5_0
```
