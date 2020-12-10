package sentinel

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

type SentinelPod struct {
	pod                *corev1.Pod
	sentinelContainerName string
	sentinelPort               int
	clientset          *kubernetes.Clientset
	restcfg            *restclient.Config
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
