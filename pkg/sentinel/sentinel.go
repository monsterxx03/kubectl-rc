package sentinel

import (
	"context"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"github.com/go-redis/redis/v8"
	"github.com/monsterxx03/kuberc/pkg/common"
)

type SentinelPod struct {
	pod                *corev1.Pod
	sentinelContainerName string
	sentinelPort               int
	clientset          *kubernetes.Clientset
	restcfg            *restclient.Config
}

type MasterInfo struct {
	Name string
	IP string
	RoleReported string
	NumSlaves int
	Flags string
}


func NewSentinelPod(podname string, sentinelContainerName string, namespace string, port int, clientset *kubernetes.Clientset, restcfg *restclient.Config) (*SentinelPod, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), podname, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if sentinelContainerName != "" {
		hasContainer := false
		for _, c := range pod.Spec.Containers {
			if c.Name == sentinelContainerName {
				hasContainer = true
			}
		}
		if !hasContainer {
			return nil, fmt.Errorf("can't find container %s in pod %s", sentinelContainerName, podname)
		}
	}
	return &SentinelPod{pod: pod, sentinelContainerName: sentinelContainerName, sentinelPort: port, clientset: clientset, restcfg: restcfg}, nil
}

func (s *SentinelPod) Info() error {
	return nil
}

func (s *SentinelPod) Masters() error {
	//if err := common.PortForward(s.clientset, s.restcfg, s.pod, s.sentinelPort, s.sentinelPort); err != nil {
	//	return err
	//}
	client := redis.NewSentinelClient(&redis.Options{Addr: fmt.Sprintf("localhost:%d", s.sentinelPort)})
	result, err := client.Masters(context.Background()).Result()
	if err != nil {
		return err
	}
	masters := parseMasters(result)
	fmt.Println(masters)
	return nil
}


func (s *SentinelPod) Failover(name string) error {
	return nil
}

func (s *SentinelPod) Check(name string) error {
	return nil
}

func (s *SentinelPod) cli(cmd string, raw bool) (string, error) {
	var c string
	if raw {
		c = fmt.Sprintf("redis-cli --raw -p %d %s", s.sentinelPort, cmd)
	} else {
		c = fmt.Sprintf("redis-cli -p %d %s", s.sentinelPort, cmd)
	}
	return s.execute(c)
}

func (s *SentinelPod) execute(cmd string) (string, error) {
	result, err := common.Execute(s.clientset, s.restcfg, &common.ExecTarget{Pod: s.pod, Container: s.sentinelContainerName}, cmd, false, false)
	if err != nil {
		return "", err
	}
	return result, nil
}


func parseSentinelResult(result []interface{}) []map[string]string {
	res := make([]map[string]string, 0, len(result))
	for _, item := range result {
		tmp := make(map[string]string)
		key := ""
		for idx, v := range item.([]interface{}) {
			if idx % 2 == 0 {
				key = v.(string)
			} else {
				tmp[key] = v.(string)
			}
		}
		res = append(res, tmp)
	}
	return res
}

func newMasterInfo(result map[string]string) *MasterInfo {
	master := new(MasterInfo)
	master.Name = result["name"]
	master.IP = result["ip"]
	master.RoleReported = result["role-reported"]
	master.NumSlaves, _ = strconv.Atoi(result["num-slaves"])
	return master
}

func parseMasters(result []interface{}) []*MasterInfo {
	masters := make([]*MasterInfo, 0, len(result))
	for _, item := range parseSentinelResult(result) {
		masters = append(masters, newMasterInfo(item))
	}
	return masters
}
