package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	"github.com/monsterxx03/kuberc/pkg/common"
)

type RedisPod struct {
	pod                *corev1.Pod
	redisContainerName string
	port               int
	nodeID             string
	clientset          *kubernetes.Clientset
	restcfg            *restclient.Config
}

func NewRedisPod(podname string, redisContainerName string, namespace string, port int, clientset *kubernetes.Clientset, restcfg *restclient.Config) (*RedisPod, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), podname, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if redisContainerName != "" {
		hasContainer := false
		for _, c := range pod.Spec.Containers {
			if c.Name == redisContainerName {
				hasContainer = true
			}
		}
		if !hasContainer {
			return nil, fmt.Errorf("can't find container %s in pod %s", redisContainerName, podname)
		}
	}
	return &RedisPod{pod: pod, redisContainerName: redisContainerName, port: port, clientset: clientset, restcfg: restcfg}, nil
}

func NewRedisPodWithPod(pod *corev1.Pod, redisContainerName string, port int, clientset *kubernetes.Clientset, restcfg *restclient.Config) *RedisPod {
	return &RedisPod{pod: pod, redisContainerName: redisContainerName, port: port, clientset: clientset, restcfg: restcfg}
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
	m, err := r.getPodsInNamespace(r.pod.Namespace)
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

func (r *RedisPod) ClusterSlots() ([]*Slots, error) {
	result, err := r.redisCliLocal("cluster slots", true)
	if err != nil {
		return nil, err
	}
	result = strings.TrimSpace(result)
	lines := strings.Split(result, "\r\n")
	if len(lines) < 5 {
		return nil, fmt.Errorf("wrong slots info %s", result)
	}
	m, err := r.getPodsInNamespace(r.pod.Namespace)
	if err != nil {
		return nil, err
	}

	newPod := func (ip, portStr, nodeID string) (*RedisPod, error) {
		p, ok := m[ip]
		if !ok {
			return nil, fmt.Errorf("cant't find pod for ip %s", ip)
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, err
		}
		pod := NewRedisPodWithPod(&p, "", port, r.clientset, r.restcfg)
		pod.nodeID = nodeID
		return pod, nil
	}

	slots := make([]*Slots, 0)
	master, err := newPod(lines[2], lines[3], lines[4])
	if err != nil {
		return nil, err
	}
	start, err := strconv.Atoi(lines[0])
	if err != nil {
		return nil, err
	}
	end, err := strconv.Atoi(lines[1])
	if err != nil {
		return nil, err
	}
	s := &Slots{Start: start, End: end, Master: master, Slaves: make([]*RedisPod, 0)}
	slots = append(slots, s)
	lines = lines[5:]
	for {
		if len(lines) == 0 {
			break
		}
		if strings.Contains(lines[0], ".") {
			// it's a slave node
			last := slots[len(slots)-1]
			slave, err := newPod(lines[0], lines[1], lines[2])
			if err != nil {
				return nil, err
			}
			last.Slaves = append(last.Slaves, slave)
			lines = lines[3:]
		} else {
			// a new slot
			master, err := newPod(lines[2], lines[3], lines[4])
			if err != nil {
				return nil, err
			}
			start, err := strconv.Atoi(lines[0])
			if err != nil {
				return nil, err
			}
			end, err := strconv.Atoi(lines[1])
			if err != nil {
				return nil, err
			}
			slots = append(slots, &Slots{Start: start, End: end, Master: master, Slaves: make([]*RedisPod, 0)})
			lines = lines[5:]
		}
	}
	return slots, nil
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
	return common.Execute(r.clientset, r.restcfg, &common.ExecTarget{Pod: r.pod, Container: r.redisContainerName}, fmt.Sprintf("redis-cli --cluster %s", cmd), toStdout, toStdin)
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
	return common.Execute(r.clientset, r.restcfg, &common.ExecTarget{Pod: r.pod, Container: r.redisContainerName}, c, false, false)
}


func (s *RedisPod) getPodsInNamespace(namespace string) (map[string]corev1.Pod, error) {
	pods, err := s.clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	m := make(map[string]corev1.Pod)
	for _, pod := range pods.Items {
		m[pod.Status.PodIP] = pod
	}
	return m, nil
}

func (p *RedisPod) getPodsInStatefulSet() (map[string]corev1.Pod, error) {
	stsName := ""
	for _, r := range p.pod.OwnerReferences {
		if *r.Controller && r.Kind == "StatefulSet" {
			stsName = r.Name
			break
		}
	}
	if stsName == "" {
		return nil, fmt.Errorf("pod %s is not managed by statefulset", p.pod.Name)
	}
	sts, err := p.clientset.AppsV1().StatefulSets(p.pod.Namespace).Get(context.Background(), stsName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	selector := make([]string, 0, len(sts.Spec.Selector.MatchLabels))
	for k, v := range sts.Spec.Selector.MatchLabels {
		selector = append(selector, fmt.Sprintf("%s=%s", k, v))
	}
	pods, err := p.clientset.CoreV1().Pods(p.pod.Namespace).List(context.Background(),
		metav1.ListOptions{LabelSelector: strings.Join(selector, ",")})
	if err != nil {
		return nil, err
	}
	m := make(map[string]corev1.Pod)
	for _, pod := range pods.Items {
		m[pod.Status.PodIP] = pod
	}
	return m, nil

}

