package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Report struct {
	Stages []*Stage `json:"stages"`
}

type Stage struct {
	Command       string `json:"command"`
	ExitCode      int    `json:"exit_code"`
	Stderr        string `json:"stderr"`
	stderrBuilder *strings.Builder
	cmd           *exec.Cmd
}

func main() {

	ctx := context.Background()

	commands := []string{}

	curCommand := ""
	for _, arg := range os.Args[1:] {
		if arg == "::" {
			commands = append(commands, curCommand)
			curCommand = ""
		} else {
			if curCommand == "" {
				curCommand = arg
			} else {
				curCommand = curCommand + " " + arg
			}
		}
	}
	commands = append(commands, curCommand)

	var nextStdout io.Reader = os.Stdin

	stages := []*Stage{}
	for _, command := range commands {
		parts := strings.Split(command, " ")
		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

		cmd.Stdin = nextStdout

		stderrBuilder := &strings.Builder{}
		cmd.Stderr = stderrBuilder

		var err error
		nextStdout, err = cmd.StdoutPipe()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		err = cmd.Start()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		stage := &Stage{
			Command:       command,
			cmd:           cmd,
			stderrBuilder: stderrBuilder,
		}

		stages = append(stages, stage)
	}

	_, err := io.Copy(os.Stdout, nextStdout)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	exitError := false
	for _, stage := range stages {
		err := stage.cmd.Wait()
		if err != nil {
			exitError = true
			stage.Stderr = stage.stderrBuilder.String()
			if exitError, ok := err.(*exec.ExitError); ok {
				stage.ExitCode = exitError.ExitCode()
			}
		} else {
			//fmt.Println("exited clean")
		}
	}

	report := Report{
		Stages: stages,
	}

	reportBytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	os.Stderr.Write(reportBytes)

	if exitError {
		os.Exit(1)
	}
}
