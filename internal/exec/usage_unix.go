//go:build unix

package exec

import (
	"os"
	"syscall"

	"github.com/loov/goda/internal/memory"
)

func TryGetUsage(state *os.ProcessState) Usage {
	usage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok {
		return Usage{}
	}

	return Usage{
		HasUsage: true,

		MaximumResidentSetSize:     memory.Bytes(usage.Maxrss),
		IntegralSharedMemorySize:   memory.Bytes(usage.Ixrss),
		IntegralUnsharedDataSize:   memory.Bytes(usage.Idrss),
		IntegralUnsharedStackSize:  memory.Bytes(usage.Isrss),
		PageReclaims:               int64(usage.Minflt),
		PageFaults:                 int64(usage.Majflt),
		Swaps:                      int64(usage.Nswap),
		BlockInputOperations:       int64(usage.Inblock),
		BlockOutputOperations:      int64(usage.Oublock),
		IPCMessagesSent:            int64(usage.Msgsnd),
		IPCMessagesReceived:        int64(usage.Msgrcv),
		SignalsReceived:            int64(usage.Nsignals),
		VoluntaryContextSwitches:   int64(usage.Nvcsw),
		InvoluntaryContextSwitches: int64(usage.Nivcsw),
	}
}
