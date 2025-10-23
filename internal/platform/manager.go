package platform

import (
	"portreleasor/internal/types"
)

// PlatformManager interface for platform-specific operations
type PlatformManager interface {
	// GetPortConnections retrieves all port connection information
	GetPortConnections() ([]types.PortInfo, error)

	// KillProcessByPID kills a process by its PID
	KillProcessByPID(pid int) error

	// GetProcessPath retrieves the process path
	GetProcessPath(pid int) (string, error)
}

// GetPlatformManager returns the platform-specific manager
var GetPlatformManager = func() PlatformManager {
	// This will return the appropriate implementation based on the platform
	// Platform-specific implementations will override this in their init() functions
	return nil
}