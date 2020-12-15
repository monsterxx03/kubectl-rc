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

// nodesCmd represents the nodes command
var nodesCmd = &cobra.Command{
	Use:   "nodes <pod>",
	Short: "List nodes in redis cluster",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := redis.NewRedisPod(args[0], containerName, namespace, redisPort, clientset, restcfg)
		if err != nil {
			return err
		}
		nodes, err := p.ClusterNodes()
		if err != nil {
			return err
		}
		masterMap := make(map[string]*redis.RedisNode)
		slaveGroups := make(map[*redis.RedisNode][]*redis.RedisNode)
		for _, n := range nodes {
			if n.IsMaster() {
				masterMap[n.ID] = n
			}
		}
		for _, n := range nodes {
			if !n.IsMaster() {
				m := masterMap[n.MasterID]
				_, ok := slaveGroups[m]
				if ok {
					slaveGroups[m] = append(slaveGroups[m], n)
				} else {
					slaveGroups[m] = []*redis.RedisNode{n}
				}
			}
		}
		for _, m := range masterMap {
			fmt.Println("Master:",masterMap[m.ID])
			if len(slaveGroups[m]) > 0 {
				for _, s := range slaveGroups[m] {
					fmt.Println("\t Slave:", s)
				}
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(nodesCmd)
}
