/*
Copyright © 2020 Will Xu <xyj.asmy@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package main

import (
	"flag"
	"fmt"
	"github.com/spf13/cobra"
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/monsterxx03/kuberc/pkg/common"
	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var cfgFile string
var sentinelNamespace string
var sentinelContainerName string
var sentinelPort int
var redisPort int
var redisContainerName string
var restcfg *restclient.Config
var clientset *kubernetes.Clientset

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sen",
	Short: "Manage redis-sentinel on k8s",
	Long:  `Used as kubectl plugin. Get redis pods monitored by redis-sentinel, to failover, replace pods.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if os.Getenv("KUBECONFIG") != "" {
			cfgFile = os.Getenv("KUBECONFIG")
		} else {
			cfgFile, err = homedir.Expand(cfgFile)
			if err != nil {
				return err
			}
		}
		restcfg, err = clientcmd.BuildConfigFromFlags("", cfgFile)
		if err != nil {
			return err
		}
		clientset, err = kubernetes.NewForConfig(restcfg)
		if err != nil {
			return err
		}
		if err := common.CheckPort(sentinelPort); err != nil {
			return err
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().IntVarP(&sentinelPort, "port", "p", 26379, "redis-sentinel port")
	rootCmd.PersistentFlags().IntVarP(&redisPort, "redis-port", "", 6379, "redis port")
	rootCmd.PersistentFlags().StringVarP(&redisContainerName, "redis-container", "", "", "redis cointainer name")
	rootCmd.PersistentFlags().StringVarP(&sentinelNamespace, "namespace", "n", "default", "sentinel pod namespace")
	rootCmd.PersistentFlags().StringVarP(&sentinelContainerName, "container", "c", "", "sentinel container name")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "~/.kube/config", "kubeconfig used for kubectl, will try to load from $KUBECONFIG first")
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlag(flag.CommandLine.Lookup("v"))
}

func main() {
	Execute()
}
