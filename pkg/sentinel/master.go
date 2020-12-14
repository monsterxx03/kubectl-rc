package sentinel

import (
	"fmt"
)

type MasterPod struct {
	RedisPod
	NumSlaves int
	Slaves    []*SlavePod
}

func (m *MasterPod) PrettyPrint() {
	fmt.Println("Master Name:", m.Name)
	fmt.Println("Master Pod:", m.Pod.Name)
	fmt.Println("IP:", m.IP)
	fmt.Println("Flags:", m.Flags)
	fmt.Println("Num Slaves", m.NumSlaves)
	fmt.Println("Slaves:")
	for _, s := range m.Slaves {
		desc, err := s.GetDescription()
		if err != nil {
			panic(err)
		}
		fmt.Println(desc)
	}
}

func (m *MasterPod) GetPodName() string {
	if m.Pod != nil {
		return m.Pod.Name
	}
	return ""
}

func (m *MasterPod) String() string {
	return fmt.Sprintf("Master<%s: %s, %s, %d slaves>", m.Name, m.GetPodName(), m.IP, m.NumSlaves)
}
