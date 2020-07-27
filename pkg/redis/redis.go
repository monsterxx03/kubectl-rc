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
	return &RedisNode{
		ID:        parts[0],
		IP:        ip,
		Flags:     flags,
		Epoch:     epoch,
		LinkState: parts[7],
	}
}

type RedisPod struct {
	pod       *corev1.Pod
	port      int
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

func (r *RedisPod) ClusterInfo() error {
	result, err := r.execute("cluster info")
	if err != nil {
		return err
	}
	fmt.Println(result)
	return nil
}

func (r *RedisPod) ClusterNodes() error {
	result, err := r.execute("cluster nodes")
	if err != nil {
		return err
	}
	m, err := getPodIPMapInNamespace(r.clientset, r.pod.Namespace)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(result, "\n") {
		if line != "" {
			node := NewRedisNode(line)
			p, ok := m[node.IP]
			if !ok {
				return errors.New("can't find pod for ip " + node.IP)
			}
			node.Pod = &p
			fmt.Println(node)
		}
	}
	return nil
}

func (r *RedisPod) execute(cmd string) (string, error) {
	req := r.clientset.CoreV1().RESTClient().Post().Resource("pods").Name(r.pod.Name).Namespace(r.pod.Namespace).SubResource("exec")
	req.VersionedParams(&corev1.PodExecOptions{
		Command: []string{"sh", "-c", fmt.Sprintf("redis-cli -p %d %s", r.port, cmd)},
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
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: buf,
		Stderr: os.Stderr,
	})
	if err != nil {
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
