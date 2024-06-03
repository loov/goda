package exec

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/subcommands"

	"github.com/loov/goda/internal/memory"
	"github.com/loov/goda/internal/templates"
)

type Command struct {
	format string
}

func (*Command) Name() string     { return "exec" }
func (*Command) Synopsis() string { return "Run command with extended statistics." }
func (*Command) Usage() string {
	return `calc <command>:
	Run command with extended statistics.

	Example:

	go build -toolexec "goda exec" .
`
}

func (cmd *Command) SetFlags(f *flag.FlagSet) {
	f.StringVar(&cmd.format, "f", "{{.Command}} {{.PackageName}} user:{{.UserTime}} system:{{.SystemTime}}{{with .MaximumResidentSetSize}} maxrss:{{.}}{{end}} in:{{.InputsSize}} out:{{.OutputSize}}", "formatting")
}

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() == 0 {
		return subcommands.ExitSuccess
	}

	args := f.Args()

	t, err := templates.Parse(cmd.format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid format string: %v\n", err)
		return subcommands.ExitFailure
	}

	command := exec.CommandContext(ctx, args[0], args[1:]...)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr

	var info Info

	startError := command.Start()
	if startError != nil {
		fmt.Fprintf(os.Stderr, "failed to start: %v\n", startError)
		return subcommands.ExitFailure
	}

	info.Start = time.Now()
	exitError := command.Wait()
	info.Finish = time.Now()
	if command.ProcessState != nil {
		info.UserTime = command.ProcessState.UserTime()
		info.SystemTime = command.ProcessState.SystemTime()

		info.Usage = TryGetUsage(command.ProcessState)
	}

	ParseArgs(&info, args)

	err = t.Execute(os.Stdout, &info)
	if err != nil {
		fmt.Fprintf(os.Stderr, "template error: %v\n", err)
	}
	fmt.Fprintln(os.Stdout)

	if exitError != nil {
		if err, ok := exitError.(*exec.ExitError); ok {
			if status, ok := err.Sys().(syscall.WaitStatus); ok {
				return subcommands.ExitStatus(status.ExitStatus())
			}
		}
		fmt.Fprintf(os.Stderr, "failed to run: %v\n", exitError)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

type Info struct {
	Command     string
	PackageName string
	Args        []string

	Usage

	Output     string
	OutputSize memory.Bytes

	Inputs     []string
	InputsSize memory.Bytes

	Start  time.Time
	Finish time.Time

	UserTime   time.Duration
	SystemTime time.Duration
}

type Usage struct {
	HasUsage bool

	MaximumResidentSetSize     memory.Bytes // maxrss
	IntegralSharedMemorySize   memory.Bytes // ixrss
	IntegralUnsharedDataSize   memory.Bytes // idrss
	IntegralUnsharedStackSize  memory.Bytes // isrss
	PageReclaims               int64        // minflt, soft page faults
	PageFaults                 int64        // majflt, hard page faults
	Swaps                      int64        // nswap
	BlockInputOperations       int64        // inblock
	BlockOutputOperations      int64        // oublock
	IPCMessagesSent            int64        // msgsnd
	IPCMessagesReceived        int64        // msgrcv
	SignalsReceived            int64        // nsignals
	VoluntaryContextSwitches   int64        // nvcsw
	InvoluntaryContextSwitches int64        // nivcsw
}

func ParseArgs(info *Info, args []string) {
	cmdname := filepath.Base(args[0])
	ext := filepath.Ext(cmdname)
	info.Command = cmdname[:len(cmdname)-len(ext)]

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "":
		case "-I", "-D", "-trimpath":
			i++
		case "-o":
			i++
			if i < len(args) {
				info.Output = args[i]
			}
		case "-p":
			i++
			if i < len(args) {
				info.PackageName = args[i]
			}
		default:
			// ignore flags
			if args[i][0] == '-' {
				continue
			}

			ext := filepath.Ext(args[i])
			if ext == ".a" || ext == ".o" || ext == ".h" || ext == ".s" || ext == ".c" || ext == ".go" {
				info.Inputs = append(info.Inputs, args[i])
			}
		}
	}

	//TODO: take into account $WORK variable
	if info.Output != "" {
		if stat, err := os.Lstat(info.Output); err == nil {
			info.OutputSize = memory.Bytes(stat.Size())
		}
	}

	for _, input := range info.Inputs {
		if stat, err := os.Lstat(input); err == nil {
			info.InputsSize += memory.Bytes(stat.Size())
		}
	}
}
