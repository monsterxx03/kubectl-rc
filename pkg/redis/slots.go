package redis

import "fmt"

type Slots struct {
	Start string
	End string
	Master *RedisPod
	Slaves []*RedisPod
}

func (s *Slots) String() string {
	if len(s.Slaves) == 0 {
		return fmt.Sprintf("%s-%s: master: %s:%d", s.Start, s.End, s.Master.pod.Name, s.Master.port)
	}
	l := fmt.Sprintf("%s-%s: master: %s:%d slaves: ", s.Start, s.End, s.Master.pod.Name, s.Master.port)
	for _, s := range s.Slaves {
		l += fmt.Sprintf("%s:%d ", s.pod.Name, s.port)
	}
	return l
}
