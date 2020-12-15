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
	"fmt"
	"github.com/monsterxx03/kuberc/pkg/redis"
	"github.com/spf13/cobra"
)

var entryPodName string

// delNodeCmd represents the delNode command
var delNodeCmd = &cobra.Command{
	Use:   "del-node <pod-to-delete>",
	Short: "Delete a node from redis cluster",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error{
		podToDelete, err := redis.NewRedisPod(args[0], containerName, namespace, redisPort, clientset, restcfg)
		if err != nil {
			return err
		}
		nodeID, err := podToDelete.GetNodeID()
		if err != nil {
			return err
		}
		entryPod := podToDelete
		if entryPodName != "" {
			entryPod, err = redis.NewRedisPod(entryPodName, containerName, namespace, redisPort, clientset, restcfg)
			if err != nil {
				return err
			}
		}
		if res, err := entryPod.ClusterDelNode(nodeID); err != nil {
			return err
		} else {
			fmt.Println(res)
		}
		return nil
	},
}

func init() {
	delNodeCmd.Flags().StringVar(&entryPodName, "entry-pod", "", "send del-node cmd to entry-pod, not target del pod")
	rootCmd.AddCommand(delNodeCmd)
}
