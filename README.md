##  kubectl plugin for managing redis cluster & and redis sentinel

- kubectl-rc: manage redis cluster on k8s
- kubectl-sen: manage sentinel based redis master/slave on k8s

### Install

Download binary from release page: https://github.com/monsterxx03/kubectl-rc/releases

- wget https://github.com/monsterxx03/kubectl-rc/releases/download/v0.1.0/kubectl-rc_darwin_amd64 -O kubectl-rc
- chmod +x kubectl-rc && sudo mv kubectl-rc /usr/local/bin/

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
          -c, --container string   container name
          -h, --help               help for rc
          -n, --namespace string   namespace (default "default")
          -p, --port int           redis port (default 6379)

kubectl sen help

    Usage:
      sen [command]

    Available Commands:
      failover    Failover redis to slave pod
      help        Help about any command
      master      Show redis master pod info
      masters     List redis masters
      restart     restart pods in sentinel sts one by one
      sync        make <slave-pod> slave of <master-pod>

    Flags:
          --config string            kubeconfig used for kubectl, will try to load from $KUBECONFIG first (default "~/.kube/config")
      -c, --container string         sentinel container name
      -h, --help                     help for sen
      -n, --namespace string         sentinel pod namespace (default "default")
      -p, --port int                 redis-sentinel port (default 26379)
          --redis-container string   redis cointainer name
          --redis-port int           redis port (default 6379)
      -v, --v Level                  number for the log level verbosity


### kubectl-rc example

Create cluster:

    >> kubectl rc create  rc-0 rc-1 rc-2 --replicas 0

Get all redis nodes:

    >> ks rc nodes rc-0

             Pod          IP                                   NodeID                       Host IsMaster Slots
            rc-0 10.0.45.194 84f62928424e945dcf56fc12f59ceead7e0101cd ip-10-0-40-50.ec2.internal     true  5461
            rc-2  10.0.43.45 96e929fbd646c8386c9587b46e3d9a58a3fcf74e ip-10-0-40-51.ec2.internal     true  5461
            rc-1  10.0.44.38 10dafd8b7c5c40f22351cdb013b16295ae722b0f ip-10-0-40-53.ec2.internal     true  5462 
    
    
Show slots info:

    >> ks rc slots  rc-0
           slots       master slaves
          0-5460         rc-0       
     10923-16383         rc-2       
      5461-10922         rc-1    

Run command on all redis nodes:

    >> kubectl rc call rc-0 get a --all


Add new redis pod `rc-3` into redis cluster as slave of `rc-0`

    >> kubectl rc add-node rc-0 rc-3 --slave

Rebalance between all redis pods:

    >> kubectl rc rebalance rc-0 --pipeline 100 --use-empty-masters

### kubectl-sen example

Show all redis masters monitored by sentinel:
    
    >> kubectl sen masters sentinel-pod-0
     Master Name    MasterPod           IP Slaves
    test-master1     tm1-0   10.0.33.11      1
    test-master2     tm2-0   10.0.31.13      1

Show detail info:

    >> kubectl sen master sentinel-pod-0 test-master1
        Master Name: test-master1
        Master Pod: tm1-0
        IP: 10.0.33.11
        Flags: master
        Num Slaves 1
        Slaves:
            Pod:ts1-0, IP:10.0.43.12, Flags:slave, LinkStatus:up, IOSecAgo:0, InSync:0
