##  kubectl plugin for managing redis cluster

Operate on k8s pods, not confusing ip and node ip in redis-cluster.

### Install

go get github.com/monsterxx03/kubectl-rc

kubectl rc help

        Usage:
          rc [command]

        Available Commands:
          add-node    Make a pod join redis-cluster
          call        Run command on redis node
          check       Check nodes for slots configuration
          create      Create redis cluster
          del-node    Delete a node from redis cluster
          failover    Promote a slave to master
          help        Help about any command
          info        Get redis cluster info
          nodes       List nodes in redis cluster
          rebalance   Rebalance slots in redis cluster
          slots       Get cluster slots info

        Flags:
              --config string      kubeconfig used for kubectl, will try to load from $KUBECONFIG first
          -h, --help               help for rc
          -n, --namespace string   namespace (default "default")
          -p, --port int           redis port (default 6379)

### Example

Create cluster:

    >> kubectl rc create  rc-0 rc-1 rc-2 --replicas 0

Get all redis nodes:

    >> ks rc nodes rc-0

    id: f1b074635d8dc1e33df8521c93a97537748099cb, ip: 10.0.45.194, host: ip-10-0-40-50.ec2.internal, pod: default/rc-0, master: true
    id: 6e008f2fdcc451fc042237691b1af5d542550252, ip: 10.0.47.232, host: ip-10-0-40-52.ec2.internal, pod: default/rc-1, master: true
    id: 0aac3566a4fee619a10bd681951c4ae26c47238d, ip: 10.0.44.165, host: ip-10-0-40-54.ec2.internal, pod: default/rc-2, master: true
    

Run command on all redis nodes:

    >> kubectl rc call rc-0 get a --all


Add new redis pod `rc-3` into redis cluster as slave of `rc-0`

    >> kubectl rc add-node rc-0 rc-3 --slave

Rebalance between all redis pods:

    >> kubectl rc rebalance rc-0 --pipeline 100 --use-empty-masters
