package main

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

func GetProcessStats(state *os.ProcessState) (info ProcessStats) {
	rusage := state.SysUsage().(*syscall.Rusage)
	info.System = state.SystemTime()
	info.Kernel = ftduration(rusage.KernelTime)
	info.Start = fttime(rusage.CreationTime)
	info.Finish = fttime(rusage.ExitTime)
	info.User = info.Finish.Sub(info.Start)

	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(state.Pid()))
	if err == nil {
		var mem ProcessMemoryCountersEx
		if err := getProcessMemoryInfo(handle, &mem); err == nil {
			info.PageFaults = int64(mem.PageFaultCount)
			info.WorkingSet = int64(mem.PeakWorkingSetSize)
			info.PagedPool = int64(mem.QuotaPeakPagedPoolUsage)
			info.NonPagedPool = int64(mem.QuotaPeakNonPagedPoolUsage)
			info.PageFile = int64(mem.PeakPagefileUsage)
		} else {
			fmt.Println("SECOND", err)
		}
	} else {
		fmt.Println("FIRST", err)
	}

	return info
}

type ProcessMemoryCountersEx struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
	PrivateUsage               uintptr
}

var (
	modpsapi                 = syscall.NewLazyDLL("psapi.dll")
	procGetProcessMemoryInfo = modpsapi.NewProc("GetProcessMemoryInfo")
)

func getProcessMemoryInfo(h syscall.Handle, mem *ProcessMemoryCountersEx) (err error) {
	r1, _, e1 := syscall.Syscall(procGetProcessMemoryInfo.Addr(), 3, uintptr(h), uintptr(unsafe.Pointer(mem)), uintptr(unsafe.Sizeof(*mem)))
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func fttime(t syscall.Filetime) time.Time {
	return time.Unix(0, t.Nanoseconds())
}

func ftduration(ft syscall.Filetime) time.Duration {
	n := int64(ft.HighDateTime)<<32 + int64(ft.LowDateTime) // in 100-nanosecond intervals
	return time.Duration(n*100) * time.Nanosecond
}
