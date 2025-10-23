package core

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"portreleasor/internal/platform"
	"portreleasor/internal/types"
	"portreleasor/internal/utils"
)

// ReleasePorts releases the specified ports by killing the processes using them
func ReleasePorts(portInputs []string, force bool) error {
	ports, err := utils.ParsePorts(portInputs)
	if err != nil {
		return err
	}

	manager := platform.GetPlatformManager()
	if manager == nil {
		return fmt.Errorf("unsupported platform")
	}

	connections, err := manager.GetPortConnections()
	if err != nil {
		return fmt.Errorf("failed to get port connections: %v", err)
	}

	portMap := make(map[int][]types.PortInfo)
	for _, conn := range connections {
		for _, port := range ports {
			if conn.Port == port {
				portMap[port] = append(portMap[port], conn)
			}
		}
	}

	if len(portMap) == 0 {
		fmt.Println("No processes found using the specified ports")
		return nil
	}

	fmt.Println("Processes using the specified ports:")
	fmt.Println("PORT/PROTOCOL\tPID\tPROCESS")
	fmt.Println("----------------------------------------")

	pidSet := make(map[int]bool)
	for _, infos := range portMap {
		for _, info := range infos {
			fmt.Println(info.String())
			pidSet[info.PID] = true
		}
	}

	if !force {
		fmt.Printf("\nKill these processes? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			// 如果无法读取输入，默认取消操作
			fmt.Println("\nOperation cancelled (could not read input)")
			return nil
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Operation cancelled")
			return nil
		}
	}

	fmt.Println("\nKilling processes...")
	successCount := 0
	failCount := 0

	for pid := range pidSet {
		if err := manager.KillProcessByPID(pid); err != nil {
			fmt.Printf("Failed to kill process %d: %v\n", pid, err)
			failCount++
		} else {
			fmt.Printf("Successfully killed process %d\n", pid)
			successCount++
		}
	}

	fmt.Printf("\nSummary: %d succeeded, %d failed\n", successCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("failed to kill %d process(es)", failCount)
	}

	return nil
}
