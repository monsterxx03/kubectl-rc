package sentinel

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type slaveSyncInfo map[string]string

type SlavePod struct {
	RedisPod
}

func (s *SlavePod) GetSyncStatus() (slaveSyncInfo, error) {
	if s.PortForwarder == nil {
		return make(slaveSyncInfo), nil
	}
	if err := s.PortForwarder.Start(); err != nil {
		return nil, err
	}
	defer s.PortForwarder.Stop()
	client := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("localhost:%d", s.Port)})
	result, err := client.Info(context.Background(), "replication").Result()
	if err != nil {
		return nil, err
	}
	return parseRedisInfo(result), nil
}

func (s *SlavePod) GetDescription() (string, error) {
	info, err := s.GetSyncStatus()
	if err != nil {
		return "", err
	}
	podName := ""
	if s.Pod != nil {
		podName = s.Pod.Name
	}
	return fmt.Sprintf("\tPod:%s, IP:%s, Flags:%s, LinkStatus:%s, IOSecAgo:%s, InSync:%s\n", podName, s.IP, s.Flags, info["master_link_status"], info["master_last_io_seconds_ago"], info["master_sync_in_progress"]), nil
}
