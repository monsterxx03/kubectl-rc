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


`kubectl rc info <pod> -n <namespace>`
    
`kubectl rc nodes <pod> -n <namespace>`

    redis-cluster-0  <node-id> 10.0.21.4 master <slots> 
    redis-cluster-1  <node-id> 10.0.21.2 <slave of redis-cluster-0> <slots> 
    ...

`kubectl rc add-slave <target-pod> <slave-pod> -n <namespace>`

`kubectl rc failover <slave-pod> -n <namespace>`
