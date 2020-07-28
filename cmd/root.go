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
package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

var cfgFile string
var namespace string
var redisPort int
var restcfg *restclient.Config
var clientset *kubernetes.Clientset

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kuberc",
	Short: "Manage redis cluster on k8s",
	Long: `Used as a kubectl plugin, to get redis cluster info, 
replace redis nodes on k8s.
`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cfgFile == "" {
			cfgFile = os.Getenv("KUBECONFIG")
			if cfgFile == "" {
				return errors.New("missing kubeconfig")
			}
		}
		var err error
		restcfg, err = clientcmd.BuildConfigFromFlags("", cfgFile)
		if err != nil {
			return err
		}
		clientset, err = kubernetes.NewForConfig(restcfg)
		if err != nil {
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
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "namespace")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "kubeconfig used for kubectl, will try to load from $KUBECONFIG first")
	rootCmd.PersistentFlags().BoolP("disable-port-forwarding", "d", false, "enable port forwarding")
}

