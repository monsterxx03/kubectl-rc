package redis

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type RedisNode struct {
	ID        string
	Pod       *corev1.Pod
	IP        string
	Flags     []string
	Epoch     int
	LinkState string
	Slots     []string
}

func (n *RedisNode) IsMaster() bool {
	for _, f := range n.Flags {
		if f == "master" {
			return true
		}
	}
	return false
}

func (n *RedisNode) String() string {
	return fmt.Sprintf("id: %s, ip: %s, host: %s, pod: %s/%s, master: %t", n.ID, n.IP, n.Pod.Spec.NodeName, n.Pod.Namespace, n.Pod.Name, n.IsMaster())
}

// https://redis.io/commands/cluster-nodes
func NewRedisNode(info string) *RedisNode {
	parts := strings.Split(info, " ")
	ip := strings.Split(parts[1], ":")[0]
	flags := strings.Split(parts[2], ",")
	epoch, _ := strconv.Atoi(parts[6])
	slots := make([]string, 0, 1)
	for _, slot := range parts[8:] {
		slots = append(slots, slot)
	}
	return &RedisNode{
		ID:        parts[0],
		IP:        ip,
		Flags:     flags,
		Epoch:     epoch,
		LinkState: parts[7],
		Slots: slots,
	}
}

type RedisPod struct {
	pod       *corev1.Pod
	port      int
	nodeID    string
	clientset *kubernetes.Clientset
	restcfg   *restclient.Config
}

func NewRedisPod(podname string, namespace string, port int, clientset *kubernetes.Clientset, restcfg *restclient.Config) (*RedisPod, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), podname, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return &RedisPod{pod: pod, port: port, clientset: clientset, restcfg: restcfg}, nil
}

func NewRedisPodWithPod(pod *corev1.Pod, port int, clientset *kubernetes.Clientset, restcfg *restclient.Config) *RedisPod {
	return &RedisPod{pod: pod, port: port, clientset: clientset, restcfg: restcfg}
}

func (r *RedisPod) GetName() string {
	return r.pod.Name
}

func (r *RedisPod) GetIP() string {
	return r.pod.Status.PodIP
}

func (r *RedisPod) ConfigGet(key string) (string, error) {
	return r.redisCli("config get " + key, false)
}

func (r *RedisPod) ConfigSet(key , value string) (string, error) {
	return r.redisCli(fmt.Sprintf("config set %s %s", key, value), false)
}

func (r *RedisPod) GetNodeID() (nodeID string, err error) {
	if r.nodeID != "" {
		return r.nodeID, nil
	}
	nodeID, err = r.redisCli("cluster myid", true)
	nodeID = strings.TrimSpace(nodeID)
	r.nodeID = nodeID
	return
}

func (r *RedisPod) isMaster() (bool, error) {
	result, err := r.redisCli("role", true)
	if err != nil {
		return  false, err
	}
	if strings.Split(result, "\r\n")[0] == "master" {
		return true, nil
	}
	return false, nil
}

func (r *RedisPod) ClusterInfo() (string, error) {
	return r.redisCli("cluster info", false)
}

func (r *RedisPod) ClusterFailover(force, takeover bool) (string, error) {
	if force && takeover {
		return "", errors.New("force and takeover can't be passed at sametime during failover")
	}
	isMaster, err := r.isMaster()
	if err != nil {
		return "", err
	}
	if isMaster {
		return "", errors.New("can't do failover on a master node")
	}
	cmd := "cluster failover"
	if force {
		cmd += " force"
	}
	if takeover {
		cmd += " takeover"
	}
	return r.redisCli(cmd, false)
}

func (r *RedisPod) ClusterRebalance(weights map[string]string, useEmptyMasters bool, timeout int, simulate bool, batch int, threshold int, replace bool) (string, error) {
	cmd := fmt.Sprintf("rebalance %s:%d", r.GetIP(), r.port)
	if weights != nil && len(weights) > 0 {
		nweights := make([]string, 0, len(weights))
		nodes, err := r.ClusterNodes()
		if err != nil {
			return "", err
		}
		m := make(map[string]string) // podName -> nodeID mapping
		for _, n := range nodes {
			m[n.Pod.Name] = n.ID
		}
		for p, w := range weights {
			if nid, ok := m[p]; ok {
				nweights = append(nweights, fmt.Sprintf("%s=%s", nid, w))
			} else {
				return "", errors.New(fmt.Sprintf("can't find pod %s in redis cluster nodes", p))
			}
		}
		cmd += " --cluster-weight " + strings.Join(nweights, " ")
	}
	if useEmptyMasters {
		cmd += " --cluster-use-empty-masters"
	}
	if timeout <=2000 {
		return "", errors.New("timeout must > 2000 ms for safety.")
	}
	cmd += fmt.Sprintf(" --cluster-timeout %d", timeout)
	if simulate {
		cmd += " --cluster-simulate"
	}
	if batch <=0 {
		return "", errors.New("pipeline size must > 0")
	}
	cmd += fmt.Sprintf(" --cluster-pipeline %d", batch)
	if threshold <= 0 {
		return "", errors.New("threshold should > 0")
	}
	cmd += fmt.Sprintf(" --cluster-threshold %d", threshold)
	if replace {
		cmd += " --cluster-replace"
	}
	return r.redisCliCluster(cmd, true)
}

func (r *RedisPod) clusterNodes() (nodes []*RedisNode, err error) {
	result, err := r.redisCli("cluster nodes", false)
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(result, "\n") {
		if line != "" {
			nodes = append(nodes, NewRedisNode(line))
		}
	}
	return
}

// ClusterNodes return redis nodes with pod info
func (r *RedisPod) ClusterNodes() (nodes []*RedisNode, err error) {
	m, err := getPodIPMapInNamespace(r.clientset, r.pod.Namespace)
	if err != nil {
		return nil, err
	}
	nodes, err = r.clusterNodes()
	if err != nil {
		return nil, err
	}
	for _, node := range nodes {
		p, ok := m[node.IP]
		if !ok {
			return nil, errors.New("can't find pod for ip " + node.IP)
		}
		node.Pod = &p
	}
	return
}

func (r *RedisPod) ClusterCheck() error {
	result, err := r.redisCliCluster(fmt.Sprintf("check %s:%d", r.GetIP(), r.port), false)
	if err != nil {
		return err
	}
	fmt.Println(result)
	return nil
}

func (r *RedisPod) ClusterSlots() (result string, err error) {
	result, err = r.redisCli("cluster slots", false)
	return
}

func (r *RedisPod) ClusterAddNode(newPod *RedisPod, slave bool) (result string, err error) {
	cmd := fmt.Sprintf("add-node %s:%d %s:%d", newPod.GetIP(), r.port, r.GetIP(), r.port)
	if slave {
		isMaster, err := r.isMaster()
		if err != nil {
			return "", err
		}
		if !isMaster {
			return "", errors.New(fmt.Sprintf("%s is not master, can't add slave for it", r.pod.Name))
		}
		nodeID, err := r.GetNodeID()
		if err != nil {
			return "", err
		}
		cmd = fmt.Sprintf("%s --cluster-slave --cluster-master-id %s", cmd, nodeID)
	}
	result, err = r.redisCliCluster(cmd, false)
	return
}

func (r *RedisPod) ClusterDelNode() (result string, err error) {
	nodeID, err := r.GetNodeID()
	if err != nil {
		return "", err
	}
	return r.redisCliCluster(fmt.Sprintf("del-node %s:%d %s", r.GetIP(), r.port, nodeID), false)
}

func (r *RedisPod) redisCliCluster(cmd string, toStdout bool) (string, error) {
	return r.execute(fmt.Sprintf("redis-cli --cluster %s", cmd), toStdout)
}

func (r *RedisPod) redisCli(cmd string, raw bool) (string, error) {
	var c string
	if raw {
		c = fmt.Sprintf("redis-cli --raw -p %d %s ", r.port, cmd)
	} else {
		c = fmt.Sprintf("redis-cli -p %d %s", r.port, cmd)
	}
	return r.execute(c, false)
}

func (r *RedisPod) execute(cmd string, toStdout bool) (string, error) {
	req := r.clientset.CoreV1().RESTClient().Post().Resource("pods").Name(r.pod.Name).Namespace(r.pod.Namespace).SubResource("exec")
	fmt.Println(cmd)
	req.VersionedParams(&corev1.PodExecOptions{
		Command: []string{"sh", "-c", cmd},
		Stdin:   false,
		Stderr:  true,
		Stdout:  true,
		TTY:     true,
	}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(r.restcfg, "POST", req.URL())
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	var opt remotecommand.StreamOptions
	if toStdout {
		opt = remotecommand.StreamOptions{
			Stdout:            os.Stdout,
			Stderr:            os.Stderr,
		}
	} else {
		opt = remotecommand.StreamOptions{
			Stdout:            buf,
			Stderr:            os.Stderr,
		}
	}
	err = exec.Stream(opt)
	if err != nil {
		fmt.Println(buf.String())
		return "", err
	}
	return buf.String(), nil
}

func getPodIPMapInNamespace(clientset *kubernetes.Clientset, namespace string) (map[string]corev1.Pod, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	m := make(map[string]corev1.Pod)
	for _, pod := range pods.Items {
		m[pod.Status.PodIP] = pod
	}
	return m, err
}

