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
	"fmt"
	"github.com/monsterxx03/kuberc/pkg/redis"
	"github.com/spf13/cobra"
)

// failoverCmd represents the failover command
var failoverCmd = &cobra.Command{
	Use:   "failover <slave-pod>",
	Short: "Promote a slave to master",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pod, err := redis.NewRedisPod(args[0], namespace, redisPort, clientset, restcfg)
		if err != nil {
			return err
		}
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			return err
		}
		takeover, err := cmd.Flags().GetBool("takeover")
		if err != nil {
			return err
		}
		if res, err := pod.ClusterFailover(force, takeover); err != nil {
			return err
		} else {
			fmt.Println(res)
		}
		return nil
	},
}

func init() {
	failoverCmd.Flags().Bool("force", false, "do manual failover without handshake with master")
	failoverCmd.Flags().Bool("takeover", false, "do manual failover without cluster consensus")
	rootCmd.AddCommand(failoverCmd)
}
