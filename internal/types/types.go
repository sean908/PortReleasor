package types

import (
	"fmt"
)

// PortInfo represents port usage information
type PortInfo struct {
	Port       int    `json:"port"`
	Protocol   string `json:"protocol"`
	PID        int    `json:"pid"`
	ProcessName string `json:"process_name"`
	ProcessPath string `json:"process_path"`
	LocalAddr  string `json:"local_addr"`
	RemoteAddr string `json:"remote_addr"`
	State      string `json:"state"`
}

// String returns the string representation of PortInfo
func (p PortInfo) String() string {
	if p.ProcessPath != "" {
		return fmt.Sprintf("%d/%s\t%d\t%s\t%s",
			p.Port, p.Protocol, p.PID, p.ProcessName, p.ProcessPath)
	}
	return fmt.Sprintf("%d/%s\t%d\t%s",
		p.Port, p.Protocol, p.PID, p.ProcessName)
}
