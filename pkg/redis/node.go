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

func (n *RedisNode) String() string {
	return fmt.Sprintf("id: %s, ip: %s, host: %s, pod: %s/%s, master: %t", n.ID, n.IP, n.Pod.Spec.NodeName, n.Pod.Namespace, n.Pod.Name, n.IsMaster())
}

// https://redis.io/commands/cluster-nodes
func NewRedisNode(info string) *RedisNode {
	parts := strings.Split(info, " ")
	ip := strings.Split(parts[1], ":")[0]
	flags := strings.Split(parts[2], ",")
	epoch, _ := strconv.Atoi(parts[6])
	slots := make([]string, 0, 1)
	for _, slot := range parts[8:] {
		slots = append(slots, slot)
	}
	return &RedisNode{
		ID:        parts[0],
		IP:        ip,
		Flags:     flags,
		Epoch:     epoch,
		LinkState: parts[7],
		Slots:     slots,
	}
}
