package redis

type Slots struct {
	Start int
	End int
	Master *RedisPod
	Slaves []*RedisPod
}
