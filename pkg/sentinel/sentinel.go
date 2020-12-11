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
	sentinelClient 		*redis.SentinelClient
	clientset          *kubernetes.Clientset
	restcfg            *restclient.Config
	podsCache    []corev1.Pod
}

type MasterInfo struct {
	Name string
	Pod *corev1.Pod
	IP string
	RoleReported string
	NumSlaves int
	Flags string
	Slaves []*SlaveInfo
}

type SlaveInfo struct {
	Name string
	Pod *corev1.Pod
	IP string
	RoleReported string
	Flags string
}

func (m *MasterInfo) PrettyPrint() {
	fmt.Println("Master Name:", m.Name)
	fmt.Println("Pod:", m.Pod.Name)
	fmt.Println("IP:", m.IP)
	fmt.Println("Flags:", m.Flags)
	fmt.Println("Slaves:")
	for _, s := range m.Slaves {
		fmt.Printf("\tPod:%s, IP:%s, Flags:%s\n", s.Pod.Name, s.IP, s.Flags)
	}
}

func (m *MasterInfo) String() string {
	pname := ""
	if m.Pod != nil {
		pname = m.Pod.Name
	}
	return fmt.Sprintf("Master<%s: %s, %s, %d slaves>", m.Name, pname, m.IP, m.NumSlaves)
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
	return &SentinelPod{pod: pod, sentinelContainerName: sentinelContainerName, sentinelPort: port,
						sentinelClient: redis.NewSentinelClient(&redis.Options{Addr: fmt.Sprintf("localhost:%d", port)}),
						clientset: clientset, restcfg: restcfg}, nil
}

func (s *SentinelPod) Info() error {
	return nil
}

func (s *SentinelPod) Masters() error {
	//if err := common.PortForward(s.clientset, s.restcfg, s.pod, s.sentinelPort, s.sentinelPort); err != nil {
	//	return err
	//}
	result, err := s.sentinelClient.Masters(context.Background()).Result()
	if err != nil {
		return err
	}
	masters, err := s.parseMasters(result)
	if err != nil {
		return err
	}
	for _, item := range masters {
		fmt.Println(item)
	}
	return nil
}

func (s *SentinelPod) Master(name string) error {
	ctx := context.Background()
	mres, err := s.sentinelClient.Master(ctx, name).Result()
	if err != nil {
		return err
	}
	m, err := s.parseMaster(mres)
	if err != nil {
		return err
	}
	sres, err := s.sentinelClient.Slaves(ctx, name).Result()
	if err != nil {
		return err
	}
	slaves, err := s.parseSlaves(sres)
	if err != nil {
		return err
	}
	m.Slaves = slaves
	m.PrettyPrint()
	return nil
}


func (s *SentinelPod) Failover(name string) error {
	return nil
}

func (s *SentinelPod) Check(name string) error {
	return nil
}

func (s *SentinelPod) getPodsInNamespace(namespace string) ([]corev1.Pod, error){
	if s.podsCache != nil {
		return s.podsCache, nil
	}
	pods, err := s.clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	s.podsCache = pods.Items
	return s.podsCache, nil
}

func (s *SentinelPod) getPodByIP(ip string) (*corev1.Pod, error) {
	pods, err := s.getPodsInNamespace(s.pod.Namespace)	
	if err != nil {
		return nil, err
	}
	for _, pod := range pods {
		if pod.Status.PodIP == ip {
			return &pod, nil
		}	
	}
	return nil, fmt.Errorf("can't find pod with ip %s", ip)
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

func (s *SentinelPod) parseMasters(result []interface{}) ([]*MasterInfo, error) {
	masters := make([]*MasterInfo, 0, len(result))
	for _, item := range parseSentinelSliceResult(result) {
		m := newMasterInfo(item)
		pod, err := s.getPodByIP(m.IP)
		if err != nil {
			return nil, err
		}
		m.Pod = pod
		masters = append(masters, m)
	}
	return masters, nil
}

func (s *SentinelPod) parseSlaves(result []interface{}) ([]*SlaveInfo, error) {
	slaves := make([]*SlaveInfo, 0, len(result))
	for _, item := range parseSentinelSliceResult(result) {
		m := newSlaveInfo(item)
		pod, err := s.getPodByIP(m.IP)
		if err != nil {
			return nil, err
		}
		m.Pod = pod
		slaves = append(slaves, m)
	}
	return slaves, nil
}

func (s *SentinelPod) parseMaster(result map[string]string) (*MasterInfo , error) {
	m := newMasterInfo(result)
	pod, err := s.getPodByIP(m.IP)
	if err != nil {
		return nil, err
	}
	m.Pod = pod
	return m, nil
}

func parseSentinelSliceResult(result []interface{}) []map[string]string {
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
	master.Flags = result["flags"]
	master.RoleReported = result["role-reported"]
	master.NumSlaves, _ = strconv.Atoi(result["num-slaves"])
	return master
}


func newSlaveInfo(result map[string]string) *SlaveInfo {
	slave := new(SlaveInfo)
	slave.Name = result["name"]
	slave.IP = result["ip"]
	slave.Flags = result["flags"]
	slave.RoleReported = result["role-reported"]
	return slave
}
