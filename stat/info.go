package main

import (
	"os"
	"path/filepath"
)

func GetCommandInfo(args []string) (info CommandInfo) {
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
			info.OutputSize = stat.Size()
		}
	}
	for _, input := range info.Inputs {
		if stat, err := os.Lstat(input); err == nil {
			info.InputsSize += stat.Size()
		}
	}

	return
}
