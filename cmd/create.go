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

var createReplicas int
// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create <pod1> <pod2> ...",
	Short: "Create redis cluster",
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		pods := make([]*redis.RedisPod, 0, len(args))
		for _, name := range args {
			p, err := redis.NewRedisPod(name, namespace, redisPort, clientset, restcfg)
			if err != nil {
				return err
			}
			pods = append(pods, p)
		}
		if res, err := pods[0].ClusterCreate(createReplicas, pods[1:]...); err != nil {
			return err
		} else {
			fmt.Println(res)
		}
		return nil
	},
}

func init() {
	createCmd.Flags().IntVar(&createReplicas,"replicas", 0, "replicas in cluster")
	rootCmd.AddCommand(createCmd)
}
