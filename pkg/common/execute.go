package common

import (
	"bytes"
	"fmt"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog/v2"
	"os"
)

type ExecTarget struct {
	Pod       *corev1.Pod
	Container string
}

func Execute(clientset *kubernetes.Clientset, restcfg *restclient.Config, target *ExecTarget, cmd string, toStdout, toStdin bool) (string, error) {
	req := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(target.Pod.Name).Namespace(target.Pod.Namespace).SubResource("exec")
	klog.V(2).Info("execute in %s: %s", target.Pod.Name, cmd)
	containerName := target.Pod.Spec.Containers[0].Name
	if target.Container != "" {
		containerName = target.Container
	}
	req.VersionedParams(&corev1.PodExecOptions{
		Container: containerName,
		Command:   []string{"sh", "-c", cmd},
		Stdin:     toStdin,
		Stderr:    true,
		Stdout:    true,
		TTY:       true,
	}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(restcfg, "POST", req.URL())
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
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: os.Stderr,
	}
	err = exec.Stream(opt)
	if err != nil {
		fmt.Println(buf.String())
		return "", err
	}
	return buf.String(), nil
}
