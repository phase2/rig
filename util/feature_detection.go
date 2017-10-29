package util

import (
	"runtime"
)

// Constants for virtualization drivers
const (
	Xhyve      = "xhyve"
	VMWare     = "vmwarefusion"
	VirtualBox = "virtualbox"
)

// SupportsNativeDocker determines if the runtime OS support docker natively,
// versus needing to run docker in a virtual machine
func SupportsNativeDocker() bool {
	return IsLinux()
}

// IsLinux detects if we are running on the linux platform
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// IsMac detects if we are running on the darwin platform
func IsMac() bool {
	return runtime.GOOS == "darwin"
}

// IsWindows detects if we are running on the microsoft windows platform
func IsWindows() bool {
	return runtime.GOOS == "windows"
}
