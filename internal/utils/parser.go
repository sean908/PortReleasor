package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// ParsePorts 解析端口参数，支持单个端口、多个端口、端口范围
func ParsePorts(ports []string) ([]int, error) {
	var result []int
	seen := make(map[int]bool)

	for _, portInput := range ports {
		if strings.Contains(portInput, "-") {
			// 处理端口范围，如 "8080-8090"
			rangePorts, err := parsePortRange(portInput)
			if err != nil {
				return nil, fmt.Errorf("无效的端口范围 '%s': %v", portInput, err)
			}
			for _, port := range rangePorts {
				if !seen[port] {
					result = append(result, port)
					seen[port] = true
				}
			}
		} else {
			// 处理单个端口
			port, err := strconv.Atoi(portInput)
			if err != nil {
				return nil, fmt.Errorf("无效的端口号 '%s': %v", portInput, err)
			}
			if port < 1 || port > 65535 {
				return nil, fmt.Errorf("端口号 %d 超出范围 (1-65535)", port)
			}
			if !seen[port] {
				result = append(result, port)
				seen[port] = true
			}
		}
	}

	return result, nil
}

// parsePortRange 解析端口范围，如 "8080-8090"
func parsePortRange(rangeStr string) ([]int, error) {
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("端口范围格式错误")
	}

	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, fmt.Errorf("无效的起始端口: %v", err)
	}

	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("无效的结束端口: %v", err)
	}

	if start < 1 || end > 65535 {
		return nil, fmt.Errorf("端口范围超出有效范围 (1-65535)")
	}

	if start > end {
		return nil, fmt.Errorf("起始端口不能大于结束端口")
	}

	var result []int
	for port := start; port <= end; port++ {
		result = append(result, port)
	}

	return result, nil
}

// MatchWildcard 检查端口是否匹配通配符模式
func MatchWildcard(port int, pattern string) bool {
	patternStr := strconv.Itoa(port)
	return strings.Contains(patternStr, pattern)
}