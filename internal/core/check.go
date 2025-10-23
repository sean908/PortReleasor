package core

import (
	"fmt"
	"strconv"
	"strings"

	"portreleasor/internal/platform"
	"portreleasor/internal/types"
	"portreleasor/internal/utils"
)

// CheckPorts checks and displays port usage information
func CheckPorts(patterns []string, verbose bool, wildcard bool) error {
	manager := platform.GetPlatformManager()
	if manager == nil {
		return fmt.Errorf("unsupported platform")
	}

	connections, err := manager.GetPortConnections()
	if err != nil {
		return fmt.Errorf("failed to get port connections: %v", err)
	}

	var filtered []types.PortInfo

	if len(patterns) == 0 {
		filtered = connections
	} else {
		for _, conn := range connections {
			for _, pattern := range patterns {
				if wildcard {
					if utils.MatchWildcard(conn.Port, pattern) {
						filtered = append(filtered, conn)
						break
					}
				} else {
					port, err := strconv.Atoi(pattern)
					if err == nil && conn.Port == port {
						filtered = append(filtered, conn)
						break
					}
				}
			}
		}
	}

	if len(filtered) == 0 {
		fmt.Println("No matching ports found")
		return nil
	}

	if verbose {
		for _, conn := range filtered {
			if conn.ProcessPath == "" {
				path, err := manager.GetProcessPath(conn.PID)
				if err == nil {
					conn.ProcessPath = path
				}
			}
		}
	}

	// 使用固定宽度格式化输出
	portProtocolWidth := 15
	pidWidth := 8
	processWidth := 20
	pathWidth := 40

	// 动态调整列宽以适应实际数据
	for _, conn := range filtered {
		portProtocol := fmt.Sprintf("%d/%s", conn.Port, conn.Protocol)
		if len(portProtocol) > portProtocolWidth {
			portProtocolWidth = len(portProtocol)
		}
		pidStr := fmt.Sprintf("%d", conn.PID)
		if len(pidStr) > pidWidth {
			pidWidth = len(pidStr)
		}
		if len(conn.ProcessName) > processWidth {
			processWidth = len(conn.ProcessName)
		}
		if verbose && len(conn.ProcessPath) > pathWidth {
			pathWidth = len(conn.ProcessPath)
		}
	}

	// 添加适当的间距
	portProtocolWidth += 2
	pidWidth += 2
	processWidth += 2
	pathWidth += 2

	// 打印表头
	if verbose {
		header := fmt.Sprintf("%-*s %-*s %-*s %s",
			portProtocolWidth, "PORT/PROTOCOL",
			pidWidth, "PID",
			processWidth, "PROCESS",
			"PATH")
		fmt.Println(header)
		fmt.Println(strings.Repeat("-", portProtocolWidth+pidWidth+processWidth+pathWidth+3))
	} else {
		header := fmt.Sprintf("%-*s %-*s %s",
			portProtocolWidth, "PORT/PROTOCOL",
			pidWidth, "PID",
			"PROCESS")
		fmt.Println(header)
		fmt.Println(strings.Repeat("-", portProtocolWidth+pidWidth+processWidth+2))
	}

	// 打印数据行
	for _, conn := range filtered {
		if verbose {
			line := fmt.Sprintf("%-*s %-*d %-*s %s",
				portProtocolWidth, fmt.Sprintf("%d/%s", conn.Port, conn.Protocol),
				pidWidth, conn.PID,
				processWidth, conn.ProcessName,
				conn.ProcessPath)
			fmt.Println(line)
		} else {
			line := fmt.Sprintf("%-*s %-*d %s",
				portProtocolWidth, fmt.Sprintf("%d/%s", conn.Port, conn.Protocol),
				pidWidth, conn.PID,
				conn.ProcessName)
			fmt.Println(line)
		}
	}

	fmt.Printf("\nShowing %d unique port(s)\n", len(filtered))

	return nil
}
