package common

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

func GetPod(podName, containerName, namespace string, clientset *kubernetes.Clientset, restcfg *restclient.Config) (*corev1.Pod, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if containerName != "" {
		hasContainer := false
		for _, c := range pod.Spec.Containers {
			if c.Name == containerName {
				hasContainer = true
			}
		}
		if !hasContainer {
			return nil, fmt.Errorf("can't find container %s in pod %s", containerName, podName)
		}
	}
	return pod, nil
}
