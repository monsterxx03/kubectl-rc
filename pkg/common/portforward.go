package common

import (
	"os"
	"os/signal"
	"syscall"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/kubernetes"
)

func PortForward(clientset *kubernetes.Clientset, restcfg *restclient.Config, pod *corev1.Pod, podPort, localPort int) error {
	if pod.Status.Phase != corev1.PodRunning {
		return fmt.Errorf("unable to forward port because pod is not running. Current status=%v", pod.Status.Phase)
	}
	stream := genericclioptions.IOStreams{
		In: os.Stdin,
		Out: os.Stdout,
		ErrOut: os.Stderr,
	}
	stopCh := make(chan struct{}, 1)
	readyCh := make(chan struct{}, 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs	
		fmt.Println("Shutdown port forwarding")
		close(stopCh)
	}()
	req := clientset.CoreV1().RESTClient().Post().Resource("pods").Namespace(pod.Namespace).Name(pod.Name).SubResource("portforward")
	transport, upgrader, err := spdy.RoundTripperFor(restcfg)
	if err != nil {
		return err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", localPort, podPort)}, stopCh, readyCh, stream.Out, stream.ErrOut)
	if err != nil {
		return err
	}
	return fw.ForwardPorts()
}
