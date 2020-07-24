## Mangement kubectl for redis cluster

- scale out redis cluster
    - add new pod (scale out sts)
    - join new pod to redis cluster
    - rebalance slots

- replace single redis pod
    - add new pod as its slave (scale out sts)
    - manually failover slave to master
    - restart old master
    - failover back to old master
    - scale in sts (added pod will be deleted)

- rolling upgrade redis pods


kubectl rc info <pod> -n <namespace>
    
kubectl rc nodes <pod> -n <namespace>

    redis-cluster-0  10.0.21.4:6379 master <slots>  <node-id>
    redis-cluster-1  10.0.21.2:6379 slave <slots>  <node-id>
    ...

kubectl rc replace <pod> -n <namespace>

kubectl rc rollout <pod> -n <namespace>

kubectl rc add-slave <target-pod> -n <namespace>
