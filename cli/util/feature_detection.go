package util

import (
  "runtime"
)

// Does the runtime OS support docker natively, versus needing to run docker in a virtual machine
func SupportsNativeDocker() bool {
  return runtime.GOOS == "linux"
}

