//go:build !unix
// +build !unix

package exec

import "os"

func TryGetUsage(state *os.ProcessState) Usage {
	return Usage{}
}
