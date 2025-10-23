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

	fmt.Println("PORT/PROTOCOL\tPID\tPROCESS" + func() string {
		if verbose {
			return "\tPATH"
		}
		return ""
	}())
	fmt.Println(strings.Repeat("-", func() int {
		if verbose {
			return 80
		}
		return 40
	}()))

	for _, conn := range filtered {
		if verbose {
			fmt.Println(conn.String())
		} else {
			fmt.Printf("%d/%s\t%d\t%s\n", conn.Port, conn.Protocol, conn.PID, conn.ProcessName)
		}
	}

	fmt.Printf("\nShowing %d unique port(s)\n", len(filtered))

	return nil
}
