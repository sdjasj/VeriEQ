package main

import (
	"VeriEQ/CodeGenerator"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
)

func (f *Fuzzer) RunIVerilog(inputData string, generator *CodeGenerator.ExpressionGenerator, realSubDir, tmpFileName, tbFileName string) ([]byte, error) {
	iverilogDir := filepath.Join(realSubDir, "iverilog")
	if err := os.MkdirAll(iverilogDir, 0755); err != nil {
		return nil, err
	}
	aoutPath := filepath.Join(iverilogDir, "a.out")
	args := []string{tmpFileName, tbFileName, "-o", aoutPath}
	cmd := exec.Command(toolConfig.IverilogPath, args...)
	cmd.Env = append(os.Environ(), "ASAN_OPTIONS=detect_leaks=0")
	cmd.Dir = realSubDir

	iverilogLogFileName := filepath.Join(realSubDir, GetRandomFileName("iverilog", ".log", ""))
	logFile, err := os.Create(iverilogLogFileName)
	defer logFile.Close()
	if err != nil {
		return nil, err
	}
	var stderrBuffer bytes.Buffer
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer

	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, iverilogLogFileName, stderrBuffer.String(), err.Error())
		return nil, err
	}

	testbenchInputPath := filepath.Join(iverilogDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(testbenchInputPath, []byte(inputData), 0644); err != nil {
		return nil, err
	}
	cmd = exec.Command("./a.out")
	cmd.Env = append(os.Environ(), "ASAN_OPTIONS=detect_leaks=0")
	cmd.Dir = iverilogDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()
	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, iverilogLogFileName, stderrBuffer.String(), err.Error())
		return nil, err
	}

	iverilogData, err := os.ReadFile(filepath.Join(iverilogDir, generator.TestBenchOutputFileName))
	if err != nil {
		handleFailure(f.CrashDir, realSubDir, iverilogLogFileName, err.Error())
		return nil, err
	}

	iverilogOut := filepath.Join(realSubDir, "iverilog_output.txt")
	_ = os.WriteFile(iverilogOut, iverilogData, 0o644)
	return iverilogData, nil
}
