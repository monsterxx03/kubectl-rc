package sentinel

import (
	"context"
	"fmt"
	"strings"
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
	sentinelPortForwarder *common.PortForwarder
	redisPort int
	clientset          *kubernetes.Clientset
	restcfg            *restclient.Config
	podsCache    []corev1.Pod
}

type RedisPod struct {
	Name string
	Port int
	Pod *corev1.Pod
	IP string
	RoleReported string
	Flags string
	PortForwarder *common.PortForwarder
}


func NewSentinelPod(sentinelPodName string, sentinelContainerName string, namespace string, sentinelPort, redisPort int, clientset *kubernetes.Clientset, restcfg *restclient.Config) (*SentinelPod, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), sentinelPodName, metav1.GetOptions{})
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
			return nil, fmt.Errorf("can't find container %s in pod %s", sentinelContainerName, sentinelPodName)
		}
	}
	forwarder, err := common.NewPortForwarder(clientset, restcfg, pod, sentinelPort, sentinelPort)
	if err != nil {
		return nil, err
	}
	return &SentinelPod{pod: pod, sentinelContainerName: sentinelContainerName, sentinelPort: sentinelPort,
						sentinelClient: redis.NewSentinelClient(&redis.Options{Addr: fmt.Sprintf("localhost:%d", sentinelPort)}),
						sentinelPortForwarder: forwarder, redisPort: redisPort,
						clientset: clientset, restcfg: restcfg}, nil
}

func (s *SentinelPod) Info() error {
	return nil
}


func (s *SentinelPod) Masters() error {
	if err := s.sentinelPortForwarder.Start(); err != nil {
		return err
	}
	defer s.sentinelPortForwarder.Stop()

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
	if err := s.sentinelPortForwarder.Start(); err != nil {
		return err
	}
	defer s.sentinelPortForwarder.Stop()

	ctx := context.Background()
	mres, err := s.sentinelClient.Master(ctx, name).Result()
	if err != nil {
		return err
	}
	m, err := s.newMasterPod(mres)
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
	if err := s.sentinelPortForwarder.Start(); err != nil {
		return err
	}
	defer s.sentinelPortForwarder.Stop()
	ctx := context.Background()
	res, err := s.sentinelClient.Failover(ctx, name).Result()
	if err != nil {
		return err
	}
	fmt.Println(res)
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

func (s *SentinelPod) parseMasters(result []interface{}) ([]*MasterPod, error) {
	masters := make([]*MasterPod, 0, len(result))
	for _, item := range parseSentinelSliceResult(result) {
		m, err := s.newMasterPod(item)
		if err != nil {
			return nil, err
		}
		masters = append(masters, m)
	}
	return masters, nil
}

func (s *SentinelPod) parseSlaves(result []interface{}) ([]*SlavePod, error) {
	slaves := make([]*SlavePod, 0, len(result))
	for _, item := range parseSentinelSliceResult(result) {
		slave, err := s.newSlavePod(item)
		if err != nil {
			return nil, err
		}
		slaves = append(slaves, slave)
	}
	return slaves, nil
}

func (s *SentinelPod) newSlavePod(result map[string]string) (slave *SlavePod, err error) {
	slave = new(SlavePod)
	slave.Port = s.redisPort
	slave.Name = result["name"]
	slave.IP = result["ip"]
	slave.Flags = result["flags"]
	slave.RoleReported = result["role-reported"]

	pod, err := s.getPodByIP(slave.IP)
	if err != nil {
		return nil, err
	}
	slave.PortForwarder, err  = common.NewPortForwarder(s.clientset, s.restcfg, pod, s.redisPort, s.redisPort)
	if err != nil {
		return nil, err
	}
	slave.Pod = pod
	return
}

func (s *SentinelPod) newMasterPod(result map[string]string) (master *MasterPod, err error) {
	master = new(MasterPod)
	master.Name = result["name"]
	master.IP = result["ip"]
	master.Flags = result["flags"]
	master.RoleReported = result["role-reported"]
	master.NumSlaves, err = strconv.Atoi(result["num-slaves"])
	pod, err := s.getPodByIP(master.IP)
	if err != nil {
		return nil, err
	}
	master.Pod = pod
	master.PortForwarder, err = common.NewPortForwarder(s.clientset, s.restcfg, pod, s.redisPort, s.redisPort)
	return
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


func parseRedisInfo(result string) (info map[string]string) {
	info = make(map[string]string)
	for _, line := range strings.Split(result, "\n") {
		parts := strings.Split(strings.TrimSpace(line), ":")
		if len(parts) >= 2 {
			info[parts[0]] = parts[1]
		}
	}
	return 
}
