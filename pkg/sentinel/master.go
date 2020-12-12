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

func (m *MasterPod) String() string {
	pname := ""
	if m.Pod != nil {
		pname = m.Pod.Name
	}
	return fmt.Sprintf("Master<%s: %s, %s, %d slaves>", m.Name, pname, m.IP, m.NumSlaves)
}
