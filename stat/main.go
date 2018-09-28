package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

type CommandInfo struct {
	Command     string
	PackageName string
	Output      string
	OutputSize  int64 // bytes
	Inputs      []string
	InputsSize  int64 // bytes
}

type ProcessStats struct {
	Start  time.Time
	Finish time.Time

	User   time.Duration
	Kernel time.Duration
	System time.Duration

	PageFaults   int64
	WorkingSet   int64
	PagedPool    int64
	NonPagedPool int64
	PageFile     int64
}

func main() {
	if len(os.Args) <= 1 {
		return
	}

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	cmd.Run()

	proc := GetProcessStats(cmd.ProcessState)
	info := GetCommandInfo(os.Args[1:])

	fmt.Print(
		"user:", proc.User.Seconds(), "\t",
		"system:", proc.System.Seconds(), "\t",
		"kernel:", proc.Kernel.Seconds(), "\t",
		"start:", proc.Start.Format(time.RFC3339), "\t",
		"finish:", proc.Finish.Format(time.RFC3339), "\t",
		"command:", info.Command, "\t",
		"package:", info.PackageName, "\t",
		"output-size:", info.OutputSize, "\t",
		"input-size:", info.InputsSize, "\t",

		"page-faults:", proc.PageFaults, "\t",
		"working-set:", proc.WorkingSet, "\t",
		"paged-pool:", proc.PagedPool, "\t",
		"non-paged-pool:", proc.NonPagedPool, "\t",
		"page-file:", proc.PageFile, "\t",

		"output:", info.Output, "\t",
		"input:", info.Inputs, "\t",
		"arguments:", os.Args[1:], "\n",
	)
}
