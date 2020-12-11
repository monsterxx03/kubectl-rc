package sentinel

import (
	"fmt"
)

type MasterPod struct {
	RedisPod
	NumSlaves int
	Slaves []*SlavePod
}

func (m *MasterPod) PrettyPrint() {
	fmt.Println("Master Name:", m.Name)
	fmt.Println("Pod:", m.Pod.Name)
	fmt.Println("IP:", m.IP)
	fmt.Println("Flags:", m.Flags)
	fmt.Println("Slaves:")
	for _, s := range m.Slaves {
		if err := s.PortForwarder.Start(); err != nil {
			panic(err)
		}
		info, err := s.GetSyncStatus()
		if err != nil {
			panic(err)
		}
		s.PortForwarder.Stop()
		fmt.Printf("\tPod:%s, IP:%s, Flags:%s, LinkStatus:%s, IOSecAgo:%s, InSync:%s\n", s.Pod.Name, s.IP, s.Flags, info["master_link_status"], info["master_last_io_seconds_ago"], info["master_sync_in_progress"])
	}
}

func (m *MasterPod) String() string {
	pname := ""
	if m.Pod != nil {
		pname = m.Pod.Name
	}
	return fmt.Sprintf("Master<%s: %s, %s, %d slaves>", m.Name, pname, m.IP, m.NumSlaves)
}

