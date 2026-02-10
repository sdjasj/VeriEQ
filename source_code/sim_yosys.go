package main

import (
	"VeriEQ/CodeGenerator"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
)

func (f *Fuzzer) RunYosysOptAndSim(inputData string, generator *CodeGenerator.ExpressionGenerator, realSubDir, tmpFileName, tbFileName string) ([]byte, []byte, error) {
	optFileName := filepath.Join(realSubDir, "opt.v")
	yosysOptFile := filepath.Join(realSubDir, "yosys_opt.log")

	logFile, err := os.Create(yosysOptFile)
	if err != nil {
		return nil, nil, err
	}
	defer logFile.Close()

	var stderrBuffer bytes.Buffer
	cmd := exec.Command(toolConfig.YosysPath, "-p", "read_verilog test.v; opt; proc; write_verilog opt.v")
	cmd.Dir = realSubDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, yosysOptFile, stderrBuffer.String()+err.Error())
		return nil, nil, err
	}

	dataNoOpt, err := f.RunIVerilog(inputData, generator, realSubDir, tmpFileName, tbFileName)
	if err != nil {
		return nil, nil, err
	}
	noOptOutPath := filepath.Join(realSubDir, "iverilog_output_noOpt.txt")
	_ = os.WriteFile(noOptOutPath, dataNoOpt, 0o644)

	optIVerilogDir := filepath.Join(realSubDir, "iverilog_opt")
	if err := os.MkdirAll(optIVerilogDir, 0755); err != nil {
		return dataNoOpt, nil, err
	}

	aoutPath := filepath.Join(optIVerilogDir, "a.out")
	args := []string{optFileName, tbFileName, "-o", aoutPath}
	cmd = exec.Command(toolConfig.IverilogPath, args...)
	cmd.Env = append(os.Environ(), "ASAN_OPTIONS=detect_leaks=0")
	cmd.Dir = realSubDir

	optLogFile := filepath.Join(realSubDir, GetRandomFileName("iverilog_opt_", ".log", ""))
	logFile, err = os.Create(optLogFile)
	if err != nil {
		return dataNoOpt, nil, err
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()

	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, optLogFile, stderrBuffer.String()+err.Error())
		return dataNoOpt, nil, err
	}

	testbenchInputPath := filepath.Join(optIVerilogDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(testbenchInputPath, []byte(inputData), 0644); err != nil {
		return dataNoOpt, nil, err
	}

	cmd = exec.Command("./a.out")
	cmd.Env = append(os.Environ(), "ASAN_OPTIONS=detect_leaks=0")
	cmd.Dir = optIVerilogDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()
	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, optLogFile, stderrBuffer.String()+err.Error())
		return dataNoOpt, nil, err
	}

	dataOpt, err := os.ReadFile(filepath.Join(optIVerilogDir, generator.TestBenchOutputFileName))
	if err != nil {
		handleFailure(f.CrashDir, realSubDir, optLogFile, err.Error())
		return dataNoOpt, nil, err
	}
	optOutPath := filepath.Join(realSubDir, "iverilog_output_Opt.txt")
	_ = os.WriteFile(optOutPath, dataOpt, 0o644)

	return dataNoOpt, dataOpt, nil
}

func (f *Fuzzer) RunYosysOpt(realSubDir, OptFileName string) error {
	var stderrBuffer bytes.Buffer
	yosysOptFile := filepath.Join(realSubDir, "yosys_opt.log")
	logFile, err := os.Create(yosysOptFile)
	if err != nil {
		return err
	}
	cmd := exec.Command(toolConfig.YosysPath, "-p", "read_verilog test.v; opt; proc; write_verilog opt.v")
	cmd.Dir = realSubDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, yosysOptFile, stderrBuffer.String()+err.Error())
		return err
	}

	optFileContent, err := os.ReadFile(OptFileName)
	if err != nil {
		return err
	}
	newContent := []byte("`timescale 1ns/1ps\n")
	newContent = append(newContent, optFileContent...)
	if err := os.WriteFile(OptFileName, newContent, 0644); err != nil {
		return err
	}
	return nil
}
