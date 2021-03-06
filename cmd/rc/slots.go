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
	"os"
	"sort"
	"strings"
	"text/tabwriter"
)

// slotsCmd represents the slots command
var slotsCmd = &cobra.Command{
	Use:   "slots <pod>",
	Short: "Get cluster slots info",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := redis.NewRedisPod(args[0], containerName, namespace, redisPort, clientset, restcfg)
		if err != nil {
			return err
		}
		if slots, err := p.ClusterSlots(); err != nil {
			return err
		} else {
			sort.Slice(slots, func(i, j int) bool {
				return slots[i].Start < slots[j].End
			})
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
			fmt.Fprintln(w, "slots\tmaster\tslaves\t")
			for _, s := range slots {
				slaves := make([]string, 0, len(s.Slaves))
				for _, slave := range s.Slaves {
					slaves = append(slaves, slave.GetName())
				}
				fmt.Fprintf(w, "%d-%d\t%s\t%s\t\n", s.Start, s.End, s.Master.GetName(), strings.Join(slaves, " "))
			}
			w.Flush()
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(slotsCmd)

}
