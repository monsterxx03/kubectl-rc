package sentinel

import (
	"fmt"
	"context"

	"github.com/go-redis/redis/v8"
)


type SlavePod struct {
	RedisPod
}

func (s *SlavePod) GetSyncStatus() (map[string]string, error) {
	client := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("localhost:%d", s.Port)})
	result, err := client.Info(context.Background(), "replication").Result()
	if err != nil {
		return nil, err
	}
	return parseRedisInfo(result), nil
}

