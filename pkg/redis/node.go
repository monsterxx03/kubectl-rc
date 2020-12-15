package redis

import (
	corev1 "k8s.io/api/core/v1"
)

import (
	"fmt"
	"strconv"
	"strings"
)

type RedisNode struct {
	ID        string
	Pod       *corev1.Pod
	IP        string
	Flags     []string
	MasterID  string
	Epoch     int
	LinkState string
	Slots     []string
}

func (n *RedisNode) IsMaster() bool {
	for _, f := range n.Flags {
		if f == "master" {
			return true
		}
	}
	return false
}

func (n *RedisNode) SlotsCount() int {
	count := 0
	for _, s := range n.Slots{
		if strings.Contains(s, "-"){
			parts := strings.Split(s, "-")
			start, _ := strconv.Atoi(parts[0])
			end, _ := strconv.Atoi(parts[1])
			count += (end - start + 1)
		} else {
			count += 1
		}
	}
	return count
}

func (n *RedisNode) String() string {
	return fmt.Sprintf("pod: %s, id: %s, ip: %s, host: %s, master: %t, slots: %d", n.Pod.Name, n.ID, n.IP, n.Pod.Spec.NodeName, n.IsMaster(), n.SlotsCount())
}

// https://redis.io/commands/cluster-nodes
func NewRedisNode(info string) *RedisNode {
	parts := strings.Split(info, " ")
	ip := strings.Split(parts[1], ":")[0]
	flags := strings.Split(parts[2], ",")
	masterID := ""
	if parts[3] != "-" {
		masterID = parts[3] 
	}
	epoch, _ := strconv.Atoi(parts[6])
	slots := make([]string, 0, 1)
	for _, slot := range parts[8:] {
		slots = append(slots, strings.TrimSpace(slot))
	}
	return &RedisNode{
		ID:        parts[0],
		IP:        ip,
		Flags:     flags,
		MasterID: masterID,
		Epoch:     epoch,
		LinkState: parts[7],
		Slots:     slots,
	}
}
