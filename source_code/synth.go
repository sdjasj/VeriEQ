package main

import (
	"VeriEQ/CodeGenerator"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

var Commands []string

func (f *Fuzzer) TestSynth() {
	generator := CodeGenerator.NewExpressionGenerator()
	generator.Name = "top"
	tmpFileName := f.TmpDir + GetRandomFileName("tmp", ".v", "")

	file, err := os.Create(tmpFileName)
	if err != nil {
		fmt.Println("Error creating Verilog file:", err)
		return
	}

	verilogCode := generator.GenerateLoopFreeModule()
	file.Write([]byte(verilogCode))
	file.Close()

	var wg sync.WaitGroup
	wg.Add(len(Commands))

	for i := 0; i < len(Commands); i++ {
		i := i
		command := Commands[i]

		go func(command string) {
			defer wg.Done()

			logFileName := f.LogDir + GetRandomFileName(command+"-", ".log", "")
			realCmd := fmt.Sprintf("read_verilog %s; %s", tmpFileName, command)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
			defer cancel()

			cmd := exec.CommandContext(ctx, toolConfig.YosysPath, "-l", logFileName, "-p", realCmd)

			var stderrBuffer bytes.Buffer
			cmd.Stderr = &stderrBuffer

			err := cmd.Run()

			exitErr, isExitErr := err.(*exec.ExitError)

			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				reason := "Yosys timeout"
				saveCrashArtifacts(f, logFileName, tmpFileName, reason)
				return
			}

			if isExitErr && exitErr.ExitCode() == 137 {
				reason := "Yosys OOM Kill"
				saveCrashArtifacts(f, logFileName, tmpFileName, reason)
				return
			}

			if err != nil {
				sanitizerError := checkSanitizerErrorFromStderr(stderrBuffer.String())

				if sanitizerError {
					appendToLogFile(logFileName, stderrBuffer.String())
					reason := "Sanitizer (ASan/UBSan)"
					saveCrashArtifacts(f, logFileName, tmpFileName, reason)
				}

			} else {
				os.Remove(logFileName)
			}
		}(command)

	}

	wg.Wait()
}

func TestOfSynth() {
	Commands = []string{
		"synth",
		"synth_achronix",
		"synth_anlogic",
		"synth_easic",
		"synth_ecp5",
		"synth_efinix",
		"synth_fabulous",
		"synth_gatemate",
		"synth_gowin",
		"synth_ice40",
		"synth_intel",
		"synth_intel_alm",
		"synth_lattice",
		"synth_nanoxplore",
		"synth_nexus",
		"synth_sf2",
		"synthprop",
		"synth_coolrunner2",
		"synth_greenpak4",
		"synth_quicklogic",
		"synth_xilinx",
	}
	fuzzer := &Fuzzer{
		StartTime: time.Now().UnixMilli(),
	}
	fuzzer.Init()
	tasks := make(chan struct{})

	for i := 0; i < 10; i++ {
		go func() {
			for range tasks {
				fuzzer.TestSynth()
			}
		}()
	}

	for {
		tasks <- struct{}{}
		// time.Sleep(time.Millisecond * 10)
	}
}
