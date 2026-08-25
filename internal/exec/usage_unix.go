//go:build unix

package exec

import (
	"os"
	"runtime"
	"syscall"

	"github.com/loov/goda/internal/memory"
)

func TryGetUsage(state *os.ProcessState) Usage {
	usage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok {
		return Usage{}
	}

	// linux reports the rss fields in kilobytes, darwin and the bsds in bytes.
	rss := memory.Bytes(1)
	if runtime.GOOS == "linux" {
		rss = 1024
	}

	return Usage{
		HasUsage: true,

		MaximumResidentSetSize:     memory.Bytes(usage.Maxrss) * rss,
		IntegralSharedMemorySize:   memory.Bytes(usage.Ixrss) * rss,
		IntegralUnsharedDataSize:   memory.Bytes(usage.Idrss) * rss,
		IntegralUnsharedStackSize:  memory.Bytes(usage.Isrss) * rss,
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
