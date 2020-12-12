package common

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/klog/v2"
)

type PortForwarder struct {
	Clientset  *kubernetes.Clientset
	RestConfig *restclient.Config
	Pod        *corev1.Pod
	LocalPort  int
	PodPort    int
	Streams    genericclioptions.IOStreams
	StopCh     chan struct{}
	ReadyCh    chan struct{}
	Started    bool
}

func NewPortForwarder(clientset *kubernetes.Clientset, restcfg *restclient.Config, pod *corev1.Pod, podPort, localPort int) *PortForwarder {
	return &PortForwarder{Clientset: clientset, RestConfig: restcfg, Pod: pod, LocalPort: localPort, PodPort: podPort,
		Streams: genericclioptions.IOStreams{In: os.Stdin, Out: ioutil.Discard, ErrOut: os.Stderr},
		StopCh:  make(chan struct{}, 1), ReadyCh: make(chan struct{}), Started: false}
}

func (p *PortForwarder) Stop() {
	if !p.Started {
		return
	}
	klog.V(2).Infof("Stop port forwarding for %s:%d->%d\n", p.Pod.Name, p.PodPort, p.LocalPort)
	close(p.StopCh)
}

func (p *PortForwarder) Start() error {
	if p.Started {
		klog.V(2).Info("port forwarding already started")
		return nil
	}
	req := p.Clientset.CoreV1().RESTClient().Post().Resource("pods").Namespace(p.Pod.Namespace).Name(p.Pod.Name).SubResource("portforward")
	transport, upgrader, err := spdy.RoundTripperFor(p.RestConfig)
	if err != nil {
		return err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", p.LocalPort, p.PodPort)}, p.StopCh, p.ReadyCh, p.Streams.Out, p.Streams.ErrOut)
	if err != nil {
		return err
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		p.Stop()
	}()
	go func() { fw.ForwardPorts() }()
	select {
	case <-p.ReadyCh:
		break
	}
	p.Started = true
	klog.V(2).Infof("Port forwarding for %s:%d->%d is ready\n", p.Pod.Name, p.PodPort, p.LocalPort)
	return nil
}

func CheckPort(port int) error {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
	if err != nil {
		return nil
	}
	conn.Close()
	return fmt.Errorf("localhost:%d is in use, can't port-forward to k8s pod", port)
}
