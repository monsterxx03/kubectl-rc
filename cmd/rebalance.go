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
	"github.com/monsterxx03/kuberc/pkg/redis"
	"github.com/spf13/cobra"
	"strings"
)

var (
	rebalanceWeight         string
	rebalanceUseEmptyMaster bool
	rebalanceTimeout        int
	rebalanceSimulate       bool
	rebalancePipeline       int
	rebalanceThreshold      int
	rebalanceReplace        bool
)

// rebalanceCmd represents the rebalance command
var rebalanceCmd = &cobra.Command{
	Use:   "rebalance <pod>",
	Short: "Rebalance slots in redis cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		weights := make(map[string]string)
		line, err := cmd.Flags().GetString("weight")
		if err != nil {
			return err
		}
		if line != "" {
			for _, w := range strings.Split(line, ",") {
				parts := strings.Split(w, "=")
				if len(parts) != 2 {
					return errors.New("wrong weight flag " + line)
				}
				weights[parts[0]] = parts[1]
			}
		}

		pod, err := redis.NewRedisPod(args[0], containerName, namespace, redisPort, clientset, restcfg)
		if err != nil {
			return err
		}
		if res, err := pod.ClusterRebalance(weights, rebalanceUseEmptyMaster, rebalanceTimeout, rebalanceSimulate, rebalancePipeline, rebalanceThreshold, rebalanceReplace); err != nil {
			return err
		} else {
			fmt.Println(res)
		}
		return nil
	},
}

func init() {
	rebalanceCmd.Flags().StringVar(&rebalanceWeight, "weight", "", "set rebalance weights for pods, eg: rc-0=1,rc-1=2")
	rebalanceCmd.Flags().BoolVar(&rebalanceUseEmptyMaster, "use-empty-masters", false, "assign slots to empty master")
	rebalanceCmd.Flags().IntVar(&rebalanceTimeout, "timeout", 60000, "migrate timeout in milliseconds in single batch")
	rebalanceCmd.Flags().BoolVar(&rebalanceSimulate,"simulate", false, "perform a dry run")
	rebalanceCmd.Flags().IntVar(&rebalancePipeline,"pipeline", 10, "migrate keys batch size")
	rebalanceCmd.Flags().IntVar(&rebalanceThreshold,"threshold", 2, "do rebalance if slots difference percentage is over threshold")
	rebalanceCmd.Flags().BoolVar(&rebalanceReplace,"replace", false, "if key existed in target node, do replace")
	rootCmd.AddCommand(rebalanceCmd)
}
