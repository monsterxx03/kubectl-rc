package redis

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

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
	return r.redisCliLocal("config get "+key, false)
}

func (r *RedisPod) Call(cmd ...string) (string, error) {
	return r.redisCliLocal(strings.Join(cmd, " "), false)
}

func (r *RedisPod) ConfigSet(key, value string) (string, error) {
	return r.redisCliLocal(fmt.Sprintf("config set %s %s", key, value), false)
}

func (r *RedisPod) Ping() (string, error) {
	return r.redisCliLocal("ping", false)
}

func (r *RedisPod) GetNodeID() (nodeID string, err error) {
	if r.nodeID != "" {
		return r.nodeID, nil
	}
	nodeID, err = r.redisCliLocal("cluster myid", true)
	nodeID = strings.TrimSpace(nodeID)
	r.nodeID = nodeID
	return
}

func (r *RedisPod) isMaster() (bool, error) {
	result, err := r.redisCliLocal("role", true)
	if err != nil {
		return false, err
	}
	if strings.Split(result, "\r\n")[0] == "master" {
		return true, nil
	}
	return false, nil
}

func (r *RedisPod) ClusterInfo() (string, error) {
	return r.redisCliLocal("cluster info", false)
}

func (r *RedisPod) ClusterCreate(replicas int, yes bool, pods ...*RedisPod) (string, error) {
	l := make([]string, 1, len(pods)+1)
	l[0] = fmt.Sprintf("%s:%d", r.GetIP(), r.port)
	for _, p := range pods {
		l = append(l, fmt.Sprintf("%s:%d", p.GetIP(), p.port))
	}
	cmd := fmt.Sprintf("create %s --cluster-replicas %d", strings.Join(l, " "), replicas)
	if yes {
		cmd += " --cluster-yes"
	}
	return r.redisCliCluster(cmd, true, !yes)
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
	return r.redisCliLocal(cmd, false)
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
	if timeout <= 2000 {
		return "", errors.New("timeout must > 2000 ms for safety.")
	}
	cmd += fmt.Sprintf(" --cluster-timeout %d", timeout)
	if simulate {
		cmd += " --cluster-simulate"
	}
	if batch <= 0 {
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
	return r.redisCliCluster(cmd, true, false)
}

func (r *RedisPod) clusterNodes() (nodes []*RedisNode, err error) {
	result, err := r.redisCliLocal("cluster nodes", false)
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
	result, err := r.redisCliCluster(fmt.Sprintf("check %s:%d", r.GetIP(), r.port), false, false)
	if err != nil {
		return err
	}
	fmt.Println(result)
	return nil
}

func (r *RedisPod) ClusterSlots() (result string, err error) {
	result, err = r.redisCliLocal("cluster slots", false)
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
	result, err = r.redisCliCluster(cmd, false, false)
	return
}

func (r *RedisPod) ClusterDelNode() (result string, err error) {
	nodeID, err := r.GetNodeID()
	if err != nil {
		return "", err
	}
	return r.redisCliCluster(fmt.Sprintf("del-node %s:%d %s", r.GetIP(), r.port, nodeID), false, false)
}

func (r *RedisPod) redisCliCluster(cmd string, toStdout, toStdin bool) (string, error) {
	return r.execute(fmt.Sprintf("redis-cli --cluster %s", cmd), toStdout, toStdin)
}

func (r *RedisPod) redisCliLocal(cmd string, raw bool) (string, error) {
	return r.redisCli(cmd, raw, "127.0.0.1", r.port)
}

func (r *RedisPod) redisCli(cmd string, raw bool, host string, port int) (string, error) {
	var c string
	if raw {
		c = fmt.Sprintf("redis-cli -c --raw -h %s -p %d %s ", host, port, cmd)
	} else {
		c = fmt.Sprintf("redis-cli -c -h %s -p %d %s", host, port, cmd)
	}
	return r.execute(c, false, false)
}

func (r *RedisPod) execute(cmd string, toStdout bool, toStdin bool) (string, error) {
	req := r.clientset.CoreV1().RESTClient().Post().Resource("pods").Name(r.pod.Name).Namespace(r.pod.Namespace).SubResource("exec")
	fmt.Println(cmd)
	req.VersionedParams(&corev1.PodExecOptions{
		Command: []string{"sh", "-c", cmd},
		Stdin:   toStdin,
		Stderr:  true,
		Stdout:  true,
		TTY:     true,
	}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(r.restcfg, "POST", req.URL())
	if err != nil {
		return "", err
	}
	var stdout io.Writer
	var stdin io.Reader
	buf := new(bytes.Buffer)
	if toStdout {
		stdout = os.Stdout
	} else {
		stdout = buf
	}
	if toStdin {
		stdin = os.Stdin
	}
	opt := remotecommand.StreamOptions{
		Stdin:             stdin,
		Stdout:            stdout,
		Stderr:            os.Stderr,
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
